package view

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	reExtends         = regexp.MustCompile(`(?i)@extends\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	reSection         = regexp.MustCompile(`(?is)@section\s*\(\s*['"]([^'"]+)['"]\s*\)(.*?)@endsection`)
	reSectionShow     = regexp.MustCompile(`(?is)@section\s*\(\s*['"]([^'"]+)['"]\s*\)(.*?)@show`)
	reYieldDefault    = regexp.MustCompile(`(?i)@yield\s*\(\s*['"]([^'"]+)['"]\s*,\s*['"]([^'"]*)['"]\s*\)`)
	reYield           = regexp.MustCompile(`(?i)@yield\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	reInclude         = regexp.MustCompile(`(?i)@include\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	reIncludeIf       = regexp.MustCompile(`(?i)@includeIf\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	reIncludeWhen     = regexp.MustCompile(`(?i)@includeWhen\s*\(\s*\$([a-zA-Z0-9_.]+)\s*,\s*['"]([^'"]+)['"]\s*(?:,\s*(\[[^\]]*\])\s*)?\)`)
	reIncludeUnless   = regexp.MustCompile(`(?i)@includeUnless\s*\(\s*\$([a-zA-Z0-9_.]+)\s*,\s*['"]([^'"]+)['"]\s*(?:,\s*(\[[^\]]*\])\s*)?\)`)
	reIncludeFirst    = regexp.MustCompile(`(?i)@includeFirst\s*\(\s*\[([^\]]+)\]\s*\)`)
	reIncludeData     = regexp.MustCompile(`(?i)@include\s*\(\s*['"]([^'"]+)['"]\s*,\s*(\[[^\]]*\])\s*\)`)
	reEach            = regexp.MustCompile(`(?i)@each\s*\(\s*['"]([^'"]+)['"]\s*,\s*\$([a-zA-Z0-9_]+)\s*,\s*['"]([^'"]+)['"]\s*(?:,\s*['"]([^'"]+)['"]\s*)?\)`)
	reOnce            = regexp.MustCompile(`(?is)@once\s*(.*?)@endonce`)
	reOnceKey         = regexp.MustCompile(`(?is)@once\s*\(\s*['"]([^'"]+)['"]\s*\)(.*?)@endonce`)
	reVerbatim        = regexp.MustCompile(`(?is)@verbatim\s*(.*?)@endverbatim`)
	reSectionShort    = regexp.MustCompile(`(?i)@section\s*\(\s*['"]([^'"]+)['"]\s*,\s*['"]([^'"]*)['"]\s*\)`)
	reSectionShortVar = regexp.MustCompile(`(?i)@section\s*\(\s*['"]([^'"]+)['"]\s*,\s*\$([a-zA-Z0-9_.]+)\s*\)`)
	rePush            = regexp.MustCompile(`(?is)@push\s*\(\s*['"]([^'"]+)['"]\s*\)(.*?)@endpush`)
	rePrepend         = regexp.MustCompile(`(?is)@prepend\s*\(\s*['"]([^'"]+)['"]\s*\)(.*?)@endprepend`)
	rePushOnce        = regexp.MustCompile(`(?is)@pushOnce\s*\(\s*['"]([^'"]+)['"]\s*(?:,\s*['"]([^'"]+)['"]\s*)?\)(.*?)@endPushOnce`)
	rePrependOnce     = regexp.MustCompile(`(?is)@prependOnce\s*\(\s*['"]([^'"]+)['"]\s*(?:,\s*['"]([^'"]+)['"]\s*)?\)(.*?)@endPrependOnce`)
	reStack           = regexp.MustCompile(`(?i)@stack\s*\(\s*['"]([^'"]+)['"]\s*(?:,\s*['"]([^'"]*)['"]\s*)?\)`)
	reComponent       = regexp.MustCompile(`(?is)@component\s*\(\s*['"]([^'"]+)['"]\s*(?:,\s*(\[[^\]]*\])\s*)?\)(.*?)@endcomponent`)
	reSlot            = regexp.MustCompile(`(?is)@slot\s*\(\s*['"]([^'"]+)['"]\s*\)(.*?)@endslot`)
	reHasSection      = regexp.MustCompile(`(?is)@hasSection\s*\(\s*['"]([^'"]+)['"]\s*\)(.*?)@endif`)
	reSectionMissing  = regexp.MustCompile(`(?is)@sectionMissing\s*\(\s*['"]([^'"]+)['"]\s*\)(.*?)@endif`)
)

type stackBag struct {
	prepend []string
	append  []string
	once    map[string]bool
}

// resolveView expands includes/extends/sections/stacks into a final template string.
func (e *Engine) resolveView(name string, seen []string) (string, error) {
	bags := map[string]*stackBag{}
	once := map[string]bool{}
	content, err := e.resolveViewWithStacks(name, seen, bags, once)
	if err != nil {
		return "", err
	}
	return applyStacks(content, bags), nil
}

func (e *Engine) resolveViewWithStacks(name string, seen []string, bags map[string]*stackBag, once map[string]bool) (string, error) {
	content, err := e.resolveViewBody(name, seen, bags, once)
	if err != nil {
		return "", err
	}
	content, err = e.expandXTags(content, seen, bags, once, nil)
	if err != nil {
		return "", err
	}
	return expandProps(content), nil
}

func (e *Engine) resolveViewBody(name string, seen []string, bags map[string]*stackBag, once map[string]bool) (string, error) {
	for _, item := range seen {
		if item == name {
			return "", fmt.Errorf("circular view dependency involving [%s]", name)
		}
	}
	seen = append(seen, name)

	raw, err := os.ReadFile(e.pathFor(name))
	if err != nil {
		return "", fmt.Errorf("view [%s] not found", name)
	}
	content := string(raw)
	content, err = e.expandIncludes(content, seen, bags, once)
	if err != nil {
		return "", err
	}
	content, err = e.expandComponents(content, seen, bags, once)
	if err != nil {
		return "", err
	}
	content, err = e.expandEach(content, seen, bags, once)
	if err != nil {
		return "", err
	}
	content = applyOnce(content, once)
	content = collectStacks(content, bags)

	if match := reExtends.FindStringSubmatch(content); len(match) == 2 {
		layoutName := match[1]
		parentSections, err := e.buildSectionParentMap(layoutName, seen, bags, once)
		if err != nil {
			return "", err
		}
		sections := applyParentToSections(extractSections(content), parentSections)
		layout, err := e.resolveRootLayout(layoutName, seen, bags, once)
		if err != nil {
			return "", err
		}
		layout = applySectionConditions(layout, sections)
		return applyYields(layout, sections), nil
	}

	return content, nil
}

func (e *Engine) expandIncludes(content string, seen []string, bags map[string]*stackBag, once map[string]bool) (string, error) {
	var firstErr error

	out := reIncludeFirst.ReplaceAllStringFunc(content, func(m string) string {
		match := reIncludeFirst.FindStringSubmatch(m)
		if len(match) != 2 {
			return m
		}
		for _, raw := range strings.Split(match[1], ",") {
			name := strings.Trim(strings.TrimSpace(raw), `'"`)
			if name == "" || !e.Exists(name) {
				continue
			}
			partial, err := e.resolveViewWithStacks(name, seen, bags, once)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return "<!-- includeFirst error: " + err.Error() + " -->"
			}
			return partial
		}
		return ""
	})
	if firstErr != nil {
		return out, firstErr
	}

	out = reIncludeWhen.ReplaceAllStringFunc(out, func(m string) string {
		match := reIncludeWhen.FindStringSubmatch(m)
		if len(match) < 3 {
			return m
		}
		cond := match[1]
		name := match[2]
		dataExpr := ""
		if len(match) >= 4 {
			dataExpr = match[3]
		}
		return e.wrapConditionalInclude(true, cond, name, dataExpr, seen, bags, once, &firstErr)
	})
	if firstErr != nil {
		return out, firstErr
	}

	out = reIncludeUnless.ReplaceAllStringFunc(out, func(m string) string {
		match := reIncludeUnless.FindStringSubmatch(m)
		if len(match) < 3 {
			return m
		}
		cond := match[1]
		name := match[2]
		dataExpr := ""
		if len(match) >= 4 {
			dataExpr = match[3]
		}
		return e.wrapConditionalInclude(false, cond, name, dataExpr, seen, bags, once, &firstErr)
	})
	if firstErr != nil {
		return out, firstErr
	}

	out = reIncludeData.ReplaceAllStringFunc(out, func(m string) string {
		match := reIncludeData.FindStringSubmatch(m)
		if len(match) != 3 {
			return m
		}
		partial, err := e.resolveViewWithStacks(match[1], seen, bags, once)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return "<!-- include error: " + err.Error() + " -->"
		}
		dictCall := parseBladeMapExpr(match[2])
		return fmt.Sprintf(`{{ with mergeDict . (%s) }}%s{{ end }}`, dictCall, partial)
	})
	if firstErr != nil {
		return out, firstErr
	}

	out = reIncludeIf.ReplaceAllStringFunc(out, func(m string) string {
		match := reIncludeIf.FindStringSubmatch(m)
		if len(match) != 2 {
			return m
		}
		if !e.Exists(match[1]) {
			return ""
		}
		partial, err := e.resolveViewWithStacks(match[1], seen, bags, once)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return "<!-- includeIf error: " + err.Error() + " -->"
		}
		return partial
	})
	if firstErr != nil {
		return out, firstErr
	}

	out = reInclude.ReplaceAllStringFunc(out, func(m string) string {
		match := reInclude.FindStringSubmatch(m)
		if len(match) != 2 {
			return m
		}
		partial, err := e.resolveViewWithStacks(match[1], seen, bags, once)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return "<!-- include error: " + err.Error() + " -->"
		}
		return partial
	})
	return out, firstErr
}

func (e *Engine) wrapConditionalInclude(when bool, cond, name, dataExpr string, seen []string, bags map[string]*stackBag, once map[string]bool, firstErr *error) string {
	partial, err := e.resolveViewWithStacks(name, seen, bags, once)
	if err != nil {
		if *firstErr == nil {
			*firstErr = err
		}
		return "<!-- include conditional error: " + err.Error() + " -->"
	}
	if dataExpr != "" {
		dictCall := parseBladeMapExpr(dataExpr)
		partial = fmt.Sprintf(`{{ with mergeDict . (%s) }}%s{{ end }}`, dictCall, partial)
	}
	if when {
		return fmt.Sprintf(`{{ if dataGet . %q }}%s{{ end }}`, cond, partial)
	}
	return fmt.Sprintf(`{{ if not (dataGet . %q) }}%s{{ end }}`, cond, partial)
}

func (e *Engine) expandComponents(content string, seen []string, bags map[string]*stackBag, once map[string]bool) (string, error) {
	var firstErr error
	out := reComponent.ReplaceAllStringFunc(content, func(m string) string {
		match := reComponent.FindStringSubmatch(m)
		if len(match) < 4 {
			return m
		}
		name := strings.TrimSpace(match[1])
		name = strings.TrimPrefix(name, "components.")
		dataExpr := match[2]
		body := match[3]

		slots := map[string]string{}
		body = reSlot.ReplaceAllStringFunc(body, func(sm string) string {
			smatch := reSlot.FindStringSubmatch(sm)
			if len(smatch) != 3 {
				return sm
			}
			slots[smatch[1]] = strings.TrimSpace(smatch[2])
			return ""
		})
		slots["slot"] = strings.TrimSpace(body)

		for key, val := range slots {
			expanded, err := e.expandXTags(val, seen, bags, once, nil)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				slots[key] = "<!-- x-component error: " + err.Error() + " -->"
				continue
			}
			slots[key] = expanded
		}

		partial, err := e.resolveViewWithStacks("components."+name, seen, bags, once)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return "<!-- component error: " + err.Error() + " -->"
		}

		args := []string{}
		if strings.TrimSpace(dataExpr) != "" {
			// flatten parseBladeMapExpr dict call into merge
			args = append(args, parseBladeMapExpr(dataExpr))
		} else {
			args = append(args, "dict")
		}
		for key, val := range slots {
			args[0] = strings.TrimSpace(args[0])
			if args[0] == "dict" {
				args[0] = fmt.Sprintf(`dict %q (safeStr %q)`, key, val)
			} else {
				args[0] = fmt.Sprintf(`%s %q (safeStr %q)`, args[0], key, val)
			}
		}
		return fmt.Sprintf(`{{ with mergeDict . (%s) }}%s{{ end }}`, args[0], partial)
	})
	return out, firstErr
}

func (e *Engine) expandEach(content string, seen []string, bags map[string]*stackBag, once map[string]bool) (string, error) {
	var firstErr error
	out := reEach.ReplaceAllStringFunc(content, func(m string) string {
		match := reEach.FindStringSubmatch(m)
		if len(match) < 4 {
			return m
		}
		partialName := match[1]
		collection := match[2]
		alias := match[3]
		emptyName := ""
		if len(match) >= 5 {
			emptyName = match[4]
		}
		partial, err := e.resolveViewWithStacks(partialName, seen, bags, once)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return "<!-- each error: " + err.Error() + " -->"
		}
		partial = rewriteNamedRangeAlias(partial, alias)
		partial = rewriteForeachParentLookups(partial, alias)
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(`{{ if not (empty (dataGet . "%s")) }}`, collection))
		sb.WriteString(fmt.Sprintf(`{{ range $__zfi, $%s := dataGet $ "%s" }}`, alias, collection))
		sb.WriteString(partial)
		sb.WriteString(`{{ end }}`)
		if emptyName != "" {
			emptyPartial, err := e.resolveViewWithStacks(emptyName, seen, bags, once)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return "<!-- each empty error: " + err.Error() + " -->"
			}
			sb.WriteString(`{{ else }}`)
			sb.WriteString(emptyPartial)
		}
		sb.WriteString(`{{ end }}`)
		return sb.String()
	})
	return out, firstErr
}

func applyOnce(content string, once map[string]bool) string {
	if once == nil {
		once = map[string]bool{}
	}
	content = reOnceKey.ReplaceAllStringFunc(content, func(m string) string {
		match := reOnceKey.FindStringSubmatch(m)
		if len(match) != 3 {
			return m
		}
		key := "once:" + match[1]
		body := strings.TrimSpace(match[2])
		if once[key] {
			return ""
		}
		once[key] = true
		return body
	})
	return reOnce.ReplaceAllStringFunc(content, func(m string) string {
		match := reOnce.FindStringSubmatch(m)
		if len(match) != 2 {
			return m
		}
		body := strings.TrimSpace(match[1])
		if once[body] {
			return ""
		}
		once[body] = true
		return body
	})
}

func collectStacks(content string, bags map[string]*stackBag) string {
	content = rePrependOnce.ReplaceAllStringFunc(content, func(m string) string {
		match := rePrependOnce.FindStringSubmatch(m)
		if len(match) < 4 {
			return m
		}
		name := match[1]
		key := match[2]
		body := strings.TrimSpace(match[3])
		if key == "" {
			key = body
		}
		bag := stackBagFor(bags, name)
		if bag.once[name+"|pre|"+key] {
			return ""
		}
		bag.once[name+"|pre|"+key] = true
		bag.prepend = append(bag.prepend, body)
		return ""
	})
	content = rePushOnce.ReplaceAllStringFunc(content, func(m string) string {
		match := rePushOnce.FindStringSubmatch(m)
		if len(match) < 4 {
			return m
		}
		name := match[1]
		key := match[2]
		body := strings.TrimSpace(match[3])
		if key == "" {
			key = body
		}
		bag := stackBagFor(bags, name)
		if bag.once[name+"|push|"+key] {
			return ""
		}
		bag.once[name+"|push|"+key] = true
		bag.append = append(bag.append, body)
		return ""
	})
	content = rePrepend.ReplaceAllStringFunc(content, func(m string) string {
		match := rePrepend.FindStringSubmatch(m)
		if len(match) != 3 {
			return m
		}
		bag := stackBagFor(bags, match[1])
		bag.prepend = append(bag.prepend, strings.TrimSpace(match[2]))
		return ""
	})
	content = rePush.ReplaceAllStringFunc(content, func(m string) string {
		match := rePush.FindStringSubmatch(m)
		if len(match) != 3 {
			return m
		}
		bag := stackBagFor(bags, match[1])
		bag.append = append(bag.append, strings.TrimSpace(match[2]))
		return ""
	})
	return content
}

func stackBagFor(bags map[string]*stackBag, name string) *stackBag {
	if bag, ok := bags[name]; ok {
		return bag
	}
	bag := &stackBag{once: map[string]bool{}}
	bags[name] = bag
	return bag
}

func applyStacks(content string, bags map[string]*stackBag) string {
	return reStack.ReplaceAllStringFunc(content, func(m string) string {
		match := reStack.FindStringSubmatch(m)
		if len(match) < 2 {
			return m
		}
		bag := bags[match[1]]
		if bag == nil {
			if len(match) >= 3 {
				return match[2]
			}
			return ""
		}
		parts := make([]string, 0, len(bag.prepend)+len(bag.append))
		parts = append(parts, bag.prepend...)
		parts = append(parts, bag.append...)
		joined := strings.Join(parts, "\n")
		if joined == "" && len(match) >= 3 {
			return match[2]
		}
		return joined
	})
}

func extractSections(content string) map[string]string {
	sections := map[string]string{}
	for _, match := range reSectionShort.FindAllStringSubmatch(content, -1) {
		if len(match) == 3 {
			sections[match[1]] = match[2]
		}
	}
	for _, match := range reSectionShortVar.FindAllStringSubmatch(content, -1) {
		if len(match) == 3 {
			// Keep Blade echo so later compile turns $var into dataGet.
			sections[match[1]] = "{{ $" + match[2] + " }}"
		}
	}
	for _, match := range reSectionShow.FindAllStringSubmatch(content, -1) {
		if len(match) == 3 {
			sections[match[1]] = strings.TrimSpace(match[2])
		}
	}
	for _, match := range reSection.FindAllStringSubmatch(content, -1) {
		if len(match) == 3 {
			sections[match[1]] = strings.TrimSpace(match[2])
		}
	}
	return sections
}

func applySectionConditions(layout string, sections map[string]string) string {
	out := reHasSection.ReplaceAllStringFunc(layout, func(m string) string {
		match := reHasSection.FindStringSubmatch(m)
		if len(match) != 3 {
			return m
		}
		if _, ok := sections[match[1]]; ok {
			return match[2]
		}
		return ""
	})
	out = reSectionMissing.ReplaceAllStringFunc(out, func(m string) string {
		match := reSectionMissing.FindStringSubmatch(m)
		if len(match) != 3 {
			return m
		}
		if _, ok := sections[match[1]]; !ok {
			return match[2]
		}
		return ""
	})
	return out
}

func applyYields(layout string, sections map[string]string) string {
	out := reYieldDefault.ReplaceAllStringFunc(layout, func(m string) string {
		match := reYieldDefault.FindStringSubmatch(m)
		if len(match) != 3 {
			return m
		}
		if value, ok := sections[match[1]]; ok {
			return value
		}
		return match[2]
	})
	out = reYield.ReplaceAllStringFunc(out, func(m string) string {
		match := reYield.FindStringSubmatch(m)
		if len(match) != 2 {
			return m
		}
		if value, ok := sections[match[1]]; ok {
			return value
		}
		return ""
	})
	// @show sections also render inline where defined in child — already extracted.
	_ = reSectionShow
	return out
}

// ViewName converts a filesystem path under the views root into a dotted name.
func (e *Engine) ViewName(path string) string {
	rel, err := filepath.Rel(e.directory, path)
	if err != nil {
		return filepath.Base(path)
	}
	rel = strings.TrimSuffix(rel, e.extension)
	return strings.ReplaceAll(filepath.ToSlash(rel), "/", ".")
}
