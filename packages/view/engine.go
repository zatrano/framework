package view

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

// Engine renders HTML views.
type Engine struct {
	mu          sync.RWMutex
	directory   string
	extension   string
	environment string
	shared      map[string]any
	funcMap     template.FuncMap
	directives  map[string]func(args string) string
	cache       map[string]*template.Template
	cacheOn     bool
	composers   []composerEntry
}

// New creates a view engine rooted at directory.
func New(directory string) *Engine {
	e := &Engine{
		directory:   directory,
		extension:   ".html",
		environment: "local",
		shared:      make(map[string]any),
		funcMap:     defaultFuncs(),
		cache:       make(map[string]*template.Template),
		cacheOn:     false,
	}
	e.funcMap["viewExists"] = func(name string) bool {
		return e.Exists(name)
	}
	return e
}

// Directory returns the views root.
func (e *Engine) Directory() string {
	return e.directory
}

// SetExtension sets the view file extension.
func (e *Engine) SetExtension(ext string) {
	e.extension = ext
}

// Share shares data across all views.
func (e *Engine) Share(key string, value any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.shared[key] = value
}

// AddFunc registers a template function.
func (e *Engine) AddFunc(name string, fn any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.funcMap[name] = fn
}

// EnableCache toggles compiled template caching.
func (e *Engine) EnableCache(enabled bool) {
	e.cacheOn = enabled
}

// ClearCache drops compiled templates from memory.
func (e *Engine) ClearCache() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cache = make(map[string]*template.Template)
}

// Exists reports whether a view exists.
func (e *Engine) Exists(name string) bool {
	_, err := os.Stat(e.pathFor(name))
	return err == nil
}

// Render renders a view to a string.
func (e *Engine) Render(name string, data map[string]any) (string, error) {
	tmpl, err := e.compile(name)
	if err != nil {
		return "", err
	}

	payload := e.mergeData(name, data)
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, payload); err != nil {
		buf.Reset()
		if err2 := tmpl.Execute(&buf, payload); err2 != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func (e *Engine) mergeData(name string, data map[string]any) map[string]any {
	e.mu.RLock()
	merged := make(map[string]any, len(e.shared)+len(data))
	for key, value := range e.shared {
		merged[key] = value
	}
	e.mu.RUnlock()

	for key, value := range data {
		merged[key] = value
	}
	e.applyComposers(name, merged)
	return merged
}

func (e *Engine) compile(name string) (*template.Template, error) {
	e.mu.RLock()
	if e.cacheOn {
		if cached, ok := e.cache[name]; ok {
			e.mu.RUnlock()
			return cached, nil
		}
	}
	funcMap := template.FuncMap{}
	for k, v := range e.funcMap {
		funcMap[k] = v
	}
	e.mu.RUnlock()
	e.bindEnvironmentFuncs(funcMap)

	resolved, err := e.resolveView(name, nil)
	if err != nil {
		return nil, err
	}

	parsed, err := e.compileBladeLike(resolved)
	if err != nil {
		return nil, fmt.Errorf("view [%s] compile error: %w", name, err)
	}
	tmpl, err := template.New(name).Funcs(funcMap).Parse(parsed)
	if err != nil {
		return nil, fmt.Errorf("view [%s] compile error: %w", name, err)
	}

	if e.cacheOn {
		e.mu.Lock()
		e.cache[name] = tmpl
		e.mu.Unlock()
	}

	return tmpl, nil
}

func (e *Engine) pathFor(name string) string {
	name = strings.ReplaceAll(name, ".", string(os.PathSeparator))
	if !strings.HasSuffix(name, e.extension) {
		name += e.extension
	}
	return filepath.Join(e.directory, name)
}

// compileBladeLike converts a small Blade-like syntax into Go templates.
func (e *Engine) compileBladeLike(input string) (string, error) {
	out := input

	// Preserve verbatim blocks from further compilation.
	verbatim := map[string]string{}
	vi := 0
	out = reVerbatim.ReplaceAllStringFunc(out, func(m string) string {
		match := reVerbatim.FindStringSubmatch(m)
		if len(match) != 2 {
			return m
		}
		key := fmt.Sprintf("@@VERBATIM_%d@@", vi)
		vi++
		verbatim[key] = match[1]
		return key
	})

	// Custom directives (after verbatim, before built-in compilation).
	out = e.applyCustomDirectives(out)

	// Strip layout directives already resolved.
	out = reExtends.ReplaceAllString(out, "")
	out = reSection.ReplaceAllString(out, "")
	out = reSectionShort.ReplaceAllString(out, "")
	out = reSectionShortVar.ReplaceAllString(out, "")
	out = reSectionShow.ReplaceAllString(out, "")
	out = reYieldDefault.ReplaceAllString(out, "")
	out = reYield.ReplaceAllString(out, "")
	out = reInclude.ReplaceAllString(out, "")
	out = reIncludeIf.ReplaceAllString(out, "")
	out = reIncludeWhen.ReplaceAllString(out, "")
	out = reIncludeUnless.ReplaceAllString(out, "")
	out = reIncludeFirst.ReplaceAllString(out, "")
	out = reIncludeData.ReplaceAllString(out, "")
	out = reEach.ReplaceAllString(out, "")
	out = reOnce.ReplaceAllString(out, "")
	out = rePushOnce.ReplaceAllString(out, "")
	out = rePrependOnce.ReplaceAllString(out, "")
	out = reComponent.ReplaceAllString(out, "")
	out = reHasSection.ReplaceAllString(out, "")
	out = reSectionMissing.ReplaceAllString(out, "")
	out = reParent.ReplaceAllString(out, "")
	out = reProps.ReplaceAllString(out, "")
	out = reAware.ReplaceAllString(out, "")

	// @php blocks are not executed; strip to a safe HTML comment.
	out = compilePhpDirectives(out)

	// Comments
	out = replaceAllRegex(out, `\{\{--.*?--\}\}`, "")

	// @foreach / @forelse must rewrite aliases before generic $var compile.
	out = compileForeachBlocks(out)
	out = compileForelseBlocks(out)

	// Attribute bag must compile before generic $var rewrite.
	out = replaceAllRegex(out, `\{\{\s*\$attributes\s*\}\}`, `{{ attributesHTML (dataGet . "attributes") }}`)
	out = replaceAllRegex(out, `\{!!\s*\$attributes\s*!!\}`, `{{ attributesHTML (dataGet . "attributes") }}`)

	// Escape-aware output with nested dotted paths.
	out = replaceAllRegex(out, `\{\{\s*\$([a-zA-Z0-9_.]+)\s*\}\}`, `{{ dataGet . "$1" }}`)
	out = replaceAllRegex(out, `\{!!\s*\$([a-zA-Z0-9_.]+)\s*!!\}`, `{{ safeStr (dataGet . "$1") }}`)

	// @json($var.path)
	out = replaceAllRegex(out, `@json\s*\(\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ json (dataGet . "$1") }}`)

	// @class / @style from map or slice vars
	out = replaceAllRegex(out, `@class\s*\(\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ classAttr (dataGet . "$1") }}`)
	out = replaceAllRegex(out, `@style\s*\(\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ styleAttr (dataGet . "$1") }}`)

	// Form boolean attributes
	for _, attr := range []string{"checked", "selected", "disabled", "readonly", "required"} {
		out = replaceAllRegex(out, `@`+attr+`\s*\(\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ attrBool (dataGet . "$1") "`+attr+`" }}`)
	}

	// @lang('key')
	out = replaceAllRegex(out, `@lang\s*\(\s*['"]([^'"]+)['"]\s*\)`, `{{ trans "$1" }}`)
	// @choice('key', $count)
	out = replaceAllRegex(out, `@choice\s*\(\s*['"]([^'"]+)['"]\s*,\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ choice "$1" (dataGet . "$2") }}`)
	// @choice('key', 3)
	out = replaceAllRegex(out, `@choice\s*\(\s*['"]([^'"]+)['"]\s*,\s*(-?[0-9]+)\s*\)`, `{{ choice "$1" $2 }}`)

	// @switch / @case / @default / @break / @endswitch
	out = compileSwitchBlocks(out)

	out = compileIfDirectives(out)
	if err := rejectUnsupportedIfDirectives(out); err != nil {
		return "", err
	}

	out = replaceAllRegex(out, `@unless\s*\(\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if not (dataGet . "$1") }}`)
	out = strings.ReplaceAll(out, "@endunless", "{{ end }}")

	out = replaceAllRegex(out, `@isset\s*\(\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if issetPath . "$1" }}`)
	out = strings.ReplaceAll(out, "@endisset", "{{ end }}")
	out = replaceAllRegex(out, `@empty\s*\(\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if empty (dataGet . "$1") }}`)
	out = strings.ReplaceAll(out, "@endempty", "{{ end }}")

	// Form / auth directives — @csrfMeta before @csrf (prefix collision).
	out = strings.ReplaceAll(out, "@csrfMeta", `<meta name="csrf-token" content="{{ dataGet . "_token" }}">`)
	out = strings.ReplaceAll(out, "@csrf", `<input type="hidden" name="_token" value="{{ dataGet . "_token" }}">`)
	out = replaceAllRegex(out, `@method\s*\(\s*['"]([^'"]+)['"]\s*\)`, `<input type="hidden" name="_method" value="$1">`)
	out = replaceAllRegex(out, `@old\s*\(\s*['"]([^'"]+)['"]\s*,\s*['"]([^'"]*)['"]\s*\)`, `{{ old . "$1" "$2" }}`)
	out = replaceAllRegex(out, `@old\s*\(\s*['"]([^'"]+)['"]\s*\)`, `{{ old . "$1" }}`)
	out = replaceAllRegex(out, `@error\s*\(\s*['"]([^'"]+)['"]\s*,\s*['"]([^'"]+)['"]\s*\)`, `{{ if hasError . "$1" "$2" }}`)
	out = replaceAllRegex(out, `@error\s*\(\s*['"]([^'"]+)['"]\s*\)`, `{{ if hasError . "$1" }}`)
	out = strings.ReplaceAll(out, "@enderror", "{{ end }}")
	out = strings.ReplaceAll(out, "@auth", `{{ if dataGet . "auth" }}`)
	out = strings.ReplaceAll(out, "@endauth", "{{ end }}")
	out = strings.ReplaceAll(out, "@guest", `{{ if dataGet . "guest" }}`)
	out = strings.ReplaceAll(out, "@endguest", "{{ end }}")

	out = compileCanDirectives(out)
	out = compileEnvDirectives(out)

	for key, body := range verbatim {
		out = strings.ReplaceAll(out, key, escapeVerbatim(body))
	}

	return restoreRangeVarTokens(out), nil
}

func compileForeachBlocks(input string) string {
	const openTag = "@foreach"
	const closeTag = "@endforeach"
	for {
		start := indexFold(input, openTag)
		if start < 0 {
			return input
		}
		headerEnd := strings.Index(input[start:], ")")
		if headerEnd < 0 {
			return input
		}
		headerEnd += start + 1
		header := input[start:headerEnd]
		reHeader := mustCompile(`(?is)@foreach\s*\(\s*\$([a-zA-Z0-9_.]+)\s+as\s+(?:\$([a-zA-Z0-9_]+)\s*=>\s*)?\$([a-zA-Z0-9_]+)\s*\)`)
		match := reHeader.FindStringSubmatch(header)
		if len(match) != 4 {
			return input
		}
		bodyStart := headerEnd
		depth := 1
		i := bodyStart
		end := -1
		for i < len(input) {
			if strings.EqualFold(substrPrefix(input, i, openTag), openTag) {
				depth++
				i += len(openTag)
				continue
			}
			if strings.EqualFold(substrPrefix(input, i, closeTag), closeTag) {
				depth--
				if depth == 0 {
					end = i
					break
				}
				i += len(closeTag)
				continue
			}
			i++
		}
		if end < 0 {
			return input
		}
		path := match[1]
		keyAlias := match[2]
		alias := match[3]
		body := input[bodyStart:end]
		var compiled string
		if keyAlias != "" {
			if len(alias) >= len(keyAlias) {
				body = rewriteNamedRangeAlias(body, alias)
				body = rewriteNamedRangeAlias(body, keyAlias)
			} else {
				body = rewriteNamedRangeAlias(body, keyAlias)
				body = rewriteNamedRangeAlias(body, alias)
			}
			body = rewriteForeachParentLookups(body, alias)
			body = compileForeachBlocks(body)
			compiled = fmt.Sprintf(`{{ $__zparent := . }}{{ range $%s, $%s := dataGet $__zparent %q }}%s{{ end }}`, keyAlias, alias, path, body)
		} else {
			// Named range alone is not enough: Go templates still set `.` to each
			// element. Capture parent and rewrite remaining root lookups.
			body = rewriteNamedRangeAlias(body, alias)
			body = rewriteForeachParentLookups(body, alias)
			body = compileForeachBlocks(body)
			compiled = fmt.Sprintf(`{{ $__zparent := . }}{{ range $__zfi, $%s := dataGet $__zparent %q }}%s{{ end }}`, alias, path, body)
		}
		input = input[:start] + compiled + input[end+len(closeTag):]
	}
}

func indexFold(s, sub string) int {
	lower := strings.ToLower(s)
	return strings.Index(lower, strings.ToLower(sub))
}

func substrPrefix(s string, i int, prefix string) string {
	if i+len(prefix) > len(s) {
		return ""
	}
	return s[i : i+len(prefix)]
}

func rewriteNamedRangeAlias(body, alias string) string {
	if alias == "" {
		return body
	}
	token := rangeVarToken(alias)
	reFieldBrace := mustCompile(`\{\{\s*\$` + regexp.QuoteMeta(alias) + `\.([a-zA-Z0-9_]+)\s*\}\}`)
	body = reFieldBrace.ReplaceAllString(body, "{{ dataGet "+token+` "$1" }}`)
	reRawField := mustCompile(`\{!!\s*\$` + regexp.QuoteMeta(alias) + `\.([a-zA-Z0-9_]+)\s*!!\}`)
	body = reRawField.ReplaceAllString(body, "{{ dataGet "+token+` "$1" }}`)
	reWhole := mustCompile(`\{\{\s*\$` + regexp.QuoteMeta(alias) + `\s*\}\}`)
	body = reWhole.ReplaceAllString(body, "{{ "+token+" }}")
	reRawWhole := mustCompile(`\{!!\s*\$` + regexp.QuoteMeta(alias) + `\s*!!\}`)
	body = reRawWhole.ReplaceAllString(body, "{{ "+token+" }}")
	// Remaining $alias.field / $alias (for @if($alias.x), comparisons, etc.)
	reAnyField := mustCompile(`\$` + regexp.QuoteMeta(alias) + `\.([a-zA-Z0-9_]+)`)
	body = reAnyField.ReplaceAllString(body, token+".$1")
	reAnyWhole := mustCompile(`\$` + regexp.QuoteMeta(alias) + `\b`)
	body = reAnyWhole.ReplaceAllString(body, token)
	return body
}

// rewriteForeachParentLookups rewrites root $vars inside a foreach body to read
// from $__zparent (captured before range), since `.` becomes the loop element.
func rewriteForeachParentLookups(body, itemAlias string) string {
	reBrace := mustCompile(`\{\{\s*\$([a-zA-Z0-9_.]+)\s*\}\}`)
	body = reBrace.ReplaceAllStringFunc(body, func(m string) string {
		match := reBrace.FindStringSubmatch(m)
		if len(match) != 2 {
			return m
		}
		path := match[1]
		if path == itemAlias || strings.HasPrefix(path, itemAlias+".") || strings.HasPrefix(path, "__ZRV_") {
			return m
		}
		return fmt.Sprintf(`{{ dataGet $__zparent %q }}`, path)
	})
	reRaw := mustCompile(`\{!!\s*\$([a-zA-Z0-9_.]+)\s*!!\}`)
	body = reRaw.ReplaceAllStringFunc(body, func(m string) string {
		match := reRaw.FindStringSubmatch(m)
		if len(match) != 2 {
			return m
		}
		path := match[1]
		if path == itemAlias || strings.HasPrefix(path, itemAlias+".") || strings.HasPrefix(path, "__ZRV_") {
			return m
		}
		return fmt.Sprintf(`{{ safeStr (dataGet $__zparent %q) }}`, path)
	})
	// Directive / comparison paths: $foo → __ZPARENT__.foo (compiled later).
	reAny := mustCompile(`\$([a-zA-Z0-9_]+(?:\.[a-zA-Z0-9_]+)*)`)
	body = reAny.ReplaceAllStringFunc(body, func(m string) string {
		path := strings.TrimPrefix(m, "$")
		if path == itemAlias || strings.HasPrefix(path, itemAlias+".") {
			return m
		}
		if strings.HasPrefix(path, "__ZRV_") || strings.HasPrefix(path, "__ZPARENT__") || path == "__zparent" || path == "__zfi" {
			return m
		}
		// Skip already rewritten mustache (no leading $ left there).
		return "__ZPARENT__." + path
	})
	// @csrf reads _token from parent scope (`.` is the loop item).
	body = strings.ReplaceAll(body, "@csrfMeta", `<meta name="csrf-token" content="{{ dataGet $__zparent "_token" }}">`)
	body = strings.ReplaceAll(body, "@csrf", `<input type="hidden" name="_token" value="{{ dataGet $__zparent "_token" }}">`)
	return body
}

func rangeVarToken(alias string) string {
	return "__ZRV_" + alias + "__"
}

func restoreRangeVarTokens(input string) string {
	re := mustCompile(`__ZRV_([a-zA-Z0-9_]+)__`)
	return re.ReplaceAllStringFunc(input, func(m string) string {
		match := re.FindStringSubmatch(m)
		if len(match) != 2 {
			return m
		}
		return "$" + match[1]
	})
}

func compileForelseBlocks(input string) string {
	re := mustCompile(`(?is)@forelse\s*\(\s*\$([a-zA-Z0-9_.]+)\s+as\s+\$([a-zA-Z0-9_]+)\s*\)(.*?)@endforelse`)
	return re.ReplaceAllStringFunc(input, func(m string) string {
		match := re.FindStringSubmatch(m)
		if len(match) != 4 {
			return m
		}
		path := match[1]
		alias := match[2]
		main, empty := splitForelseEmpty(match[3])
		main = rewriteNamedRangeAlias(main, alias)
		main = rewriteForeachParentLookups(main, alias)
		var b strings.Builder
		b.WriteString(fmt.Sprintf(`{{ if not (empty (dataGet . %q)) }}{{ $__zparent := . }}{{ range $__zfi, $%s := dataGet $__zparent %q }}%s{{ end }}{{ else }}%s{{ end }}`, path, alias, path, main, empty))
		return b.String()
	})
}

func splitForelseEmpty(body string) (main, empty string) {
	idx := 0
	for {
		i := strings.Index(body[idx:], "@empty")
		if i < 0 {
			return strings.TrimSpace(body), ""
		}
		i += idx
		rest := strings.TrimSpace(body[i+len("@empty"):])
		if strings.HasPrefix(rest, "(") {
			idx = i + len("@empty")
			continue
		}
		return strings.TrimSpace(body[:i]), strings.TrimSpace(body[i+len("@empty"):])
	}
}

func compileSwitchBlocks(input string) string {
	re := mustCompile(`(?is)@switch\s*\(\s*\$([a-zA-Z0-9_.]+)\s*\)(.*?)@endswitch`)
	return re.ReplaceAllStringFunc(input, func(m string) string {
		match := re.FindStringSubmatch(m)
		if len(match) != 3 {
			return m
		}
		path := match[1]
		body := match[2]
		body = strings.ReplaceAll(body, "@break", "")
		var b strings.Builder
		first := true
		caseRe := mustCompile(`(?is)@case\s*\(\s*(['"][^'"]*['"]|[0-9]+)\s*\)`)
		defaultRe := mustCompile(`(?is)@default`)
		// Split by @case / @default while preserving markers.
		tokens := mustCompile(`(?is)(@case\s*\(\s*(?:['"][^'"]*['"]|[0-9]+)\s*\)|@default)`).Split(body, -1)
		markers := mustCompile(`(?is)(@case\s*\(\s*(?:['"][^'"]*['"]|[0-9]+)\s*\)|@default)`).FindAllString(body, -1)
		if len(markers) == 0 {
			return ""
		}
		for i, marker := range markers {
			content := ""
			if i+1 < len(tokens) {
				content = strings.TrimSpace(tokens[i+1])
			}
			if defaultRe.MatchString(marker) {
				if first {
					b.WriteString(`{{ if false }}`)
					first = false
				}
				b.WriteString(`{{ else }}`)
				b.WriteString(content)
				continue
			}
			cm := caseRe.FindStringSubmatch(marker)
			if len(cm) != 2 {
				continue
			}
			raw := strings.TrimSpace(cm[1])
			var cmp string
			if strings.HasPrefix(raw, "'") || strings.HasPrefix(raw, `"`) {
				cmp = fmt.Sprintf("%q", strings.Trim(raw, `"'`))
			} else {
				cmp = raw
			}
			if first {
				b.WriteString(fmt.Sprintf(`{{ if eq (printf "%%v" (dataGet . %q)) (printf "%%v" %s) }}`, path, cmp))
				first = false
			} else {
				b.WriteString(fmt.Sprintf(`{{ else if eq (printf "%%v" (dataGet . %q)) (printf "%%v" %s) }}`, path, cmp))
			}
			b.WriteString(content)
		}
		b.WriteString(`{{ end }}`)
		return b.String()
	})
}

func escapeVerbatim(body string) string {
	body = strings.ReplaceAll(body, "{{", "\x00OB\x00")
	body = strings.ReplaceAll(body, "}}", "\x00CB\x00")
	body = strings.ReplaceAll(body, "\x00OB\x00", "{{`{{`}}")
	body = strings.ReplaceAll(body, "\x00CB\x00", "{{`}}`}}")
	return body
}

func replaceAllRegex(input, pattern, repl string) string {
	re := mustCompile(pattern)
	return re.ReplaceAllString(input, repl)
}

func compileIfDirectives(out string) string {
	type rule struct {
		pattern string
		repl    string
	}
	// More specific operators first (>= before >, etc.).
	rules := []rule{
		// Parent-scope field (foreach) vs quoted string / number / inequalities / other vars
		{`@if\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*==\s*['"]([^'"]*)['"]\s*\)`, `{{ if eq (printf "%v" (dataGet $__zparent "$1")) "$2" }}`},
		{`@if\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*!=\s*['"]([^'"]*)['"]\s*\)`, `{{ if ne (printf "%v" (dataGet $__zparent "$1")) "$2" }}`},
		{`@if\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*>=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if cmpGe (dataGet $__zparent "$1") $2 }}`},
		{`@if\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*<=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if cmpLe (dataGet $__zparent "$1") $2 }}`},
		{`@if\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*>\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if cmpGt (dataGet $__zparent "$1") $2 }}`},
		{`@if\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*<\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if cmpLt (dataGet $__zparent "$1") $2 }}`},
		{`@if\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*==\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if eq (printf "%v" (dataGet $__zparent "$1")) "$2" }}`},
		{`@if\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*!=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if ne (printf "%v" (dataGet $__zparent "$1")) "$2" }}`},
		{`@if\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*>=\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*\)`, `{{ if cmpGe (dataGet $__zparent "$1") (dataGet $__zparent "$2") }}`},
		{`@if\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*<=\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*\)`, `{{ if cmpLe (dataGet $__zparent "$1") (dataGet $__zparent "$2") }}`},
		{`@if\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*>\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*\)`, `{{ if cmpGt (dataGet $__zparent "$1") (dataGet $__zparent "$2") }}`},
		{`@if\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*<\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*\)`, `{{ if cmpLt (dataGet $__zparent "$1") (dataGet $__zparent "$2") }}`},
		{`@if\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*\)`, `{{ if dataGet $__zparent "$1" }}`},
		{`@elseif\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*==\s*['"]([^'"]*)['"]\s*\)`, `{{ else if eq (printf "%v" (dataGet $__zparent "$1")) "$2" }}`},
		{`@elseif\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*!=\s*['"]([^'"]*)['"]\s*\)`, `{{ else if ne (printf "%v" (dataGet $__zparent "$1")) "$2" }}`},
		{`@elseif\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*>=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if cmpGe (dataGet $__zparent "$1") $2 }}`},
		{`@elseif\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*<=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if cmpLe (dataGet $__zparent "$1") $2 }}`},
		{`@elseif\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*>\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if cmpGt (dataGet $__zparent "$1") $2 }}`},
		{`@elseif\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*<\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if cmpLt (dataGet $__zparent "$1") $2 }}`},
		{`@elseif\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*==\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if eq (printf "%v" (dataGet $__zparent "$1")) "$2" }}`},
		{`@elseif\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*!=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if ne (printf "%v" (dataGet $__zparent "$1")) "$2" }}`},
		{`@elseif\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*>=\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*\)`, `{{ else if cmpGe (dataGet $__zparent "$1") (dataGet $__zparent "$2") }}`},
		{`@elseif\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*<=\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*\)`, `{{ else if cmpLe (dataGet $__zparent "$1") (dataGet $__zparent "$2") }}`},
		{`@elseif\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*>\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*\)`, `{{ else if cmpGt (dataGet $__zparent "$1") (dataGet $__zparent "$2") }}`},
		{`@elseif\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*<\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*\)`, `{{ else if cmpLt (dataGet $__zparent "$1") (dataGet $__zparent "$2") }}`},
		{`@elseif\s*\(\s*__ZPARENT__\.([a-zA-Z0-9_.]+)\s*\)`, `{{ else if dataGet $__zparent "$1" }}`},

		// Range-var field vs quoted string
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*==\s*['"]([^'"]*)['"]\s*\)`, `{{ if eq (printf "%v" (dataGet $$1 "$2")) "$3" }}`},
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*!=\s*['"]([^'"]*)['"]\s*\)`, `{{ if ne (printf "%v" (dataGet $$1 "$2")) "$3" }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*==\s*['"]([^'"]*)['"]\s*\)`, `{{ else if eq (printf "%v" (dataGet $$1 "$2")) "$3" }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*!=\s*['"]([^'"]*)['"]\s*\)`, `{{ else if ne (printf "%v" (dataGet $$1 "$2")) "$3" }}`},

		// Range-var field vs number / inequalities
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*>=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if cmpGe (dataGet $$1 "$2") $3 }}`},
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*<=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if cmpLe (dataGet $$1 "$2") $3 }}`},
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*>\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if cmpGt (dataGet $$1 "$2") $3 }}`},
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*<\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if cmpLt (dataGet $$1 "$2") $3 }}`},
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*==\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if eq (printf "%v" (dataGet $$1 "$2")) "$3" }}`},
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*!=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if ne (printf "%v" (dataGet $$1 "$2")) "$3" }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*>=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if cmpGe (dataGet $$1 "$2") $3 }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*<=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if cmpLe (dataGet $$1 "$2") $3 }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*>\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if cmpGt (dataGet $$1 "$2") $3 }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*<\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if cmpLt (dataGet $$1 "$2") $3 }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*==\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if eq (printf "%v" (dataGet $$1 "$2")) "$3" }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*!=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if ne (printf "%v" (dataGet $$1 "$2")) "$3" }}`},

		// Range-var field vs other $var
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*>=\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if cmpGe (dataGet $$1 "$2") (dataGet . "$3") }}`},
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*<=\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if cmpLe (dataGet $$1 "$2") (dataGet . "$3") }}`},
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*>\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if cmpGt (dataGet $$1 "$2") (dataGet . "$3") }}`},
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*<\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if cmpLt (dataGet $$1 "$2") (dataGet . "$3") }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*>=\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ else if cmpGe (dataGet $$1 "$2") (dataGet . "$3") }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*<=\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ else if cmpLe (dataGet $$1 "$2") (dataGet . "$3") }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*>\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ else if cmpGt (dataGet $$1 "$2") (dataGet . "$3") }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*<\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ else if cmpLt (dataGet $$1 "$2") (dataGet . "$3") }}`},

		// Root $var vs quoted string / number / inequalities / other $var
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*==\s*['"]([^'"]*)['"]\s*\)`, `{{ if eq (printf "%v" (dataGet . "$1")) "$2" }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*!=\s*['"]([^'"]*)['"]\s*\)`, `{{ if ne (printf "%v" (dataGet . "$1")) "$2" }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*>=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if cmpGe (dataGet . "$1") $2 }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*<=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if cmpLe (dataGet . "$1") $2 }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*>\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if cmpGt (dataGet . "$1") $2 }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*<\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if cmpLt (dataGet . "$1") $2 }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*==\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if eq (printf "%v" (dataGet . "$1")) "$2" }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*!=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ if ne (printf "%v" (dataGet . "$1")) "$2" }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*>=\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if cmpGe (dataGet . "$1") (dataGet . "$2") }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*<=\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if cmpLe (dataGet . "$1") (dataGet . "$2") }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*>\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if cmpGt (dataGet . "$1") (dataGet . "$2") }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*<\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if cmpLt (dataGet . "$1") (dataGet . "$2") }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*==\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if eq (printf "%v" (dataGet . "$1")) (printf "%v" (dataGet . "$2")) }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*!=\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if ne (printf "%v" (dataGet . "$1")) (printf "%v" (dataGet . "$2")) }}`},

		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*==\s*['"]([^'"]*)['"]\s*\)`, `{{ else if eq (printf "%v" (dataGet . "$1")) "$2" }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*!=\s*['"]([^'"]*)['"]\s*\)`, `{{ else if ne (printf "%v" (dataGet . "$1")) "$2" }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*>=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if cmpGe (dataGet . "$1") $2 }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*<=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if cmpLe (dataGet . "$1") $2 }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*>\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if cmpGt (dataGet . "$1") $2 }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*<\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if cmpLt (dataGet . "$1") $2 }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*==\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if eq (printf "%v" (dataGet . "$1")) "$2" }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*!=\s*(-?[0-9]+(?:\.[0-9]+)?)\s*\)`, `{{ else if ne (printf "%v" (dataGet . "$1")) "$2" }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*>=\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ else if cmpGe (dataGet . "$1") (dataGet . "$2") }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*<=\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ else if cmpLe (dataGet . "$1") (dataGet . "$2") }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*>\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ else if cmpGt (dataGet . "$1") (dataGet . "$2") }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*<\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ else if cmpLt (dataGet . "$1") (dataGet . "$2") }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*==\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ else if eq (printf "%v" (dataGet . "$1")) (printf "%v" (dataGet . "$2")) }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*!=\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ else if ne (printf "%v" (dataGet . "$1")) (printf "%v" (dataGet . "$2")) }}`},

		// Truthiness
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*\)`, `{{ if dataGet $$1 "$2" }}`},
		{`@if\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\s*\)`, `{{ if $$1 }}`},
		{`@if\s*\(\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ if dataGet . "$1" }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\.([a-zA-Z0-9_]+)\s*\)`, `{{ else if dataGet $$1 "$2" }}`},
		{`@elseif\s*\(\s*__ZRV_([a-zA-Z0-9_]+)__\s*\)`, `{{ else if $$1 }}`},
		{`@elseif\s*\(\s*\$([a-zA-Z0-9_.]+)\s*\)`, `{{ else if dataGet . "$1" }}`},
	}
	for _, r := range rules {
		out = replaceAllRegex(out, r.pattern, r.repl)
	}
	// @else must not match @elseif (already consumed).
	out = replaceAllRegex(out, `@else\b`, `{{ else }}`)
	out = strings.ReplaceAll(out, "@endif", "{{ end }}")
	return out
}

func rejectUnsupportedIfDirectives(out string) error {
	re := mustCompile(`(?i)@(?:else)?if\s*\(`)
	loc := re.FindStringIndex(out)
	if loc == nil {
		return nil
	}
	end := loc[1] + 80
	if end > len(out) {
		end = len(out)
	}
	snippet := strings.TrimSpace(out[loc[0]:end])
	snippet = strings.ReplaceAll(snippet, "\n", " ")
	return fmt.Errorf("unsupported @if/@elseif expression (supported: truthiness, ==, !=, >, >=, <, <=): %s", snippet)
}

func defaultFuncs() template.FuncMap {
	return template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"trim":  strings.TrimSpace,
		"join":  strings.Join,
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeStr":       safeStr,
		"dataGet":       dataGet,
		"json":          toJSON,
		"cmpGt":         cmpGt,
		"cmpGe":         cmpGe,
		"cmpLt":         cmpLt,
		"cmpLe":         cmpLe,
		"classAttr":     classAttr,
		"styleAttr":     styleAttr,
		"attrBool":      attrBool,
		"dict":          dict,
		"mergeDict":     mergeDict,
		"mergeDefaults": mergeDefaults,
		"isset": func(data map[string]any, key string) bool {
			if data == nil {
				return false
			}
			_, ok := data[key]
			return ok
		},
		"issetPath": func(data map[string]any, path string) bool {
			if data == nil || path == "" {
				return false
			}
			parts := strings.Split(path, ".")
			if len(parts) == 1 {
				_, ok := data[parts[0]]
				return ok
			}
			parent := dataGet(data, strings.Join(parts[:len(parts)-1], "."))
			if parent == nil {
				return false
			}
			key := parts[len(parts)-1]
			switch m := parent.(type) {
			case map[string]any:
				_, ok := m[key]
				return ok
			case map[string]string:
				_, ok := m[key]
				return ok
			default:
				return dataGet(data, path) != nil
			}
		},
		"empty": isEmptyValue,
		"old": func(data map[string]any, key string, fallback ...string) string {
			if data == nil {
				if len(fallback) > 0 {
					return fallback[0]
				}
				return ""
			}
			if old, ok := data["old"].(map[string]string); ok {
				if value, exists := old[key]; exists {
					return value
				}
			}
			if len(fallback) > 0 {
				return fallback[0]
			}
			return ""
		},
		"hasError": func(data map[string]any, key string, bagName ...string) bool {
			return hasErrorIn(lookupErrorBag(data, bagName...), key)
		},
		"error": func(data map[string]any, key string, bagName ...string) string {
			return errorIn(lookupErrorBag(data, bagName...), key)
		},
		"trans": func(key string) string {
			return key
		},
		"can": func(data map[string]any, ability string, args ...any) bool {
			if data == nil {
				return false
			}
			if fn, ok := data["__can"].(func(string, ...any) bool); ok && fn != nil {
				return fn(ability, args...)
			}
			return false
		},
		"attributesBag":  attributesBag,
		"attributesHTML": attributesHTML,
		"env": func(name string) bool {
			return name == "local"
		},
		"production": func() bool {
			return false
		},
		"viewExists": func(name string) bool {
			return false
		},
	}
}

func lookupErrorBag(data map[string]any, bagName ...string) any {
	if data == nil {
		return nil
	}
	if len(bagName) > 0 && bagName[0] != "" {
		name := bagName[0]
		if bags, ok := data["errorBags"].(map[string]any); ok {
			if bag, exists := bags[name]; exists {
				return bag
			}
		}
		if bag, ok := data[name]; ok {
			return bag
		}
		return nil
	}
	return data["errors"]
}

func hasErrorIn(bag any, key string) bool {
	switch b := bag.(type) {
	case interface{ Has(string) bool }:
		return b.Has(key)
	case map[string]string:
		return b[key] != ""
	case map[string][]string:
		return len(b[key]) > 0
	default:
		return false
	}
}

func errorIn(bag any, key string) string {
	switch b := bag.(type) {
	case interface{ First(string) string }:
		return b.First(key)
	case map[string]string:
		return b[key]
	case map[string][]string:
		if len(b[key]) > 0 {
			return b[key][0]
		}
	}
	return ""
}

func (e *Engine) bindEnvironmentFuncs(funcMap template.FuncMap) {
	env := e.environmentName()
	funcMap["env"] = func(name string) bool {
		return env == name
	}
	funcMap["production"] = func() bool {
		return env == "production"
	}
}

func isEmptyValue(v any) bool {
	if v == nil {
		return true
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x) == ""
	case bool:
		return !x
	case int:
		return x == 0
	case int64:
		return x == 0
	case float64:
		return x == 0
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Map, reflect.Array, reflect.Chan:
		return rv.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return rv.IsNil()
	}
	return false
}
