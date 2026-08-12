package view

import (
	"fmt"
	"html"
	"html/template"
	"regexp"
	"sort"
	"strings"
)

// AttrBag holds HTML attributes for component attribute bags.
type AttrBag map[string]string

// String renders attributes as an HTML attribute string.
func (a AttrBag) String() string {
	if len(a) == 0 {
		return ""
	}
	keys := make([]string, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := a[k]
		if v == "" {
			parts = append(parts, html.EscapeString(k))
			continue
		}
		parts = append(parts, fmt.Sprintf(`%s="%s"`, html.EscapeString(k), html.EscapeString(v)))
	}
	return strings.Join(parts, " ")
}

// HTML returns the attribute bag as safe HTML for {!! $attributes !!}.
func (a AttrBag) HTML() template.HTML {
	return template.HTML(a.String())
}

// Get returns an attribute value.
func (a AttrBag) Get(key string, fallback ...string) string {
	if a == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	if v, ok := a[key]; ok {
		return v
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

// Has reports whether an attribute exists.
func (a AttrBag) Has(key string) bool {
	if a == nil {
		return false
	}
	_, ok := a[key]
	return ok
}

// Except returns a copy without the given keys.
func (a AttrBag) Except(keys ...string) AttrBag {
	skip := map[string]bool{}
	for _, k := range keys {
		skip[k] = true
	}
	out := AttrBag{}
	for k, v := range a {
		if !skip[k] {
			out[k] = v
		}
	}
	return out
}

// Only returns a copy with only the given keys.
func (a AttrBag) Only(keys ...string) AttrBag {
	out := AttrBag{}
	for _, k := range keys {
		if v, ok := a[k]; ok {
			out[k] = v
		}
	}
	return out
}

// MergeClass adds classes onto the class attribute.
func (a AttrBag) MergeClass(extra string) AttrBag {
	out := AttrBag{}
	for k, v := range a {
		out[k] = v
	}
	extra = strings.TrimSpace(extra)
	if extra == "" {
		return out
	}
	if cur := strings.TrimSpace(out["class"]); cur != "" {
		out["class"] = cur + " " + extra
	} else {
		out["class"] = extra
	}
	return out
}

var (
	reXTagStart  = regexp.MustCompile(`(?is)<x-([a-zA-Z0-9_.-]+)((?:\s[^>]*?)?)\s*(/>|>)`)
	reXSlotNamed = regexp.MustCompile(`(?is)<x-slot\s*:?\s*name\s*=\s*['"]([^'"]+)['"]\s*>(.*?)</x-slot>`)
	reXSlotShort = regexp.MustCompile(`(?is)<x-slot:([a-zA-Z0-9_-]+)>(.*?)</x-slot(?::[a-zA-Z0-9_-]+)?>`)
	reAware      = regexp.MustCompile(`(?i)@aware\s*\(\s*\[([^\]]*)\]\s*\)`)
	reAttrPair   = regexp.MustCompile(`([:@]?[a-zA-Z_:][\w:.-]*)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	reBoolAttr   = regexp.MustCompile(`^[a-zA-Z_:][\w:.-]*$`)
)

func (e *Engine) expandXTags(content string, seen []string, bags map[string]*stackBag, once map[string]bool, parentProps map[string]string) (string, error) {
	var firstErr error
	replace := func(name, attrStr, body string) string {
		rendered, err := e.renderXComponent(name, attrStr, body, seen, bags, once, parentProps)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return "<!-- x-component error: " + err.Error() + " -->"
		}
		return rendered
	}

	// Expand innermost tags first so nesting works without backreferences (RE2).
	for {
		start, end, name, attrs, body, ok := findNextXTag(content)
		if !ok {
			break
		}
		content = content[:start] + replace(name, attrs, body) + content[end:]
	}
	return content, firstErr
}

func findNextXTag(content string) (start, end int, name, attrs, body string, ok bool) {
	bestStart, bestEnd := -1, -1
	bestName, bestAttrs, bestBody := "", "", ""
	search := content
	offset := 0
	for {
		loc := reXTagStart.FindStringSubmatchIndex(search)
		if loc == nil {
			break
		}
		absStart := offset + loc[0]
		tagName := search[loc[2]:loc[3]]
		tagAttrs := search[loc[4]:loc[5]]
		closer := search[loc[6]:loc[7]]
		absAfterOpen := offset + loc[1]

		if closer == "/>" {
			span := absAfterOpen - absStart
			if bestStart < 0 || span < bestEnd-bestStart {
				bestStart, bestEnd = absStart, absAfterOpen
				bestName, bestAttrs, bestBody = tagName, tagAttrs, ""
			}
			offset = absAfterOpen
			search = content[offset:]
			continue
		}

		closePat := "</x-" + tagName + ">"
		rest := content[absAfterOpen:]
		depth := 1
		pos := 0
		lowerRest := strings.ToLower(rest)
		lowerClose := strings.ToLower(closePat)
		lowerOpen := strings.ToLower("<x-" + tagName)
		for depth > 0 {
			nextClose := strings.Index(lowerRest[pos:], lowerClose)
			nextOpen := indexOpenTag(lowerRest[pos:], lowerOpen)
			if nextClose < 0 {
				break
			}
			if nextOpen >= 0 && nextOpen < nextClose {
				pos += nextOpen + len(lowerOpen)
				depth++
				continue
			}
			pos += nextClose
			depth--
			if depth == 0 {
				bodyStart := absAfterOpen
				bodyEnd := absAfterOpen + pos
				closeEnd := bodyEnd + len(closePat)
				span := closeEnd - absStart
				if bestStart < 0 || span < bestEnd-bestStart {
					bestStart, bestEnd = absStart, closeEnd
					bestName, bestAttrs = tagName, tagAttrs
					bestBody = content[bodyStart:bodyEnd]
				}
				break
			}
			pos += len(closePat)
		}
		offset = absAfterOpen
		search = content[offset:]
	}
	if bestStart < 0 {
		return 0, 0, "", "", "", false
	}
	return bestStart, bestEnd, bestName, bestAttrs, bestBody, true
}

func indexOpenTag(lower, lowerOpenPrefix string) int {
	pos := 0
	for {
		i := strings.Index(lower[pos:], lowerOpenPrefix)
		if i < 0 {
			return -1
		}
		abs := pos + i
		after := abs + len(lowerOpenPrefix)
		if after >= len(lower) {
			return abs
		}
		ch := lower[after]
		// Ensure we matched the full tag name boundary: next is whitespace, >, or /.
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '>' || ch == '/' {
			return abs
		}
		// Longer name like x-alert-extra; keep scanning.
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' || ch == '_' {
			pos = after
			continue
		}
		return abs
	}
}

func (e *Engine) renderXComponent(name, attrStr, body string, seen []string, bags map[string]*stackBag, once map[string]bool, parentProps map[string]string) (string, error) {
	viewName := xComponentViewName(name)
	attrs := parseHTMLAttributes(attrStr)

	slots := map[string]string{}
	body = reXSlotShort.ReplaceAllStringFunc(body, func(m string) string {
		match := reXSlotShort.FindStringSubmatch(m)
		if len(match) != 3 {
			return m
		}
		slots[match[1]] = strings.TrimSpace(match[2])
		return ""
	})
	body = reXSlotNamed.ReplaceAllStringFunc(body, func(m string) string {
		match := reXSlotNamed.FindStringSubmatch(m)
		if len(match) != 3 {
			return m
		}
		slots[match[1]] = strings.TrimSpace(match[2])
		return ""
	})
	slots["slot"] = strings.TrimSpace(body)

	partial, err := e.resolveViewBody(viewName, seen, bags, once)
	if err != nil {
		return "", err
	}

	// Apply @aware from parent props.
	awareKeys := extractAwareKeys(partial)
	partial = reAware.ReplaceAllString(partial, "")
	for _, key := range awareKeys {
		if _, exists := attrs[key]; !exists {
			if v, ok := parentProps[key]; ok {
				attrs[key] = v
			}
		}
	}

	propKeys := extractPropsKeys(partial)
	propAttrs := AttrBag{}
	bagAttrs := AttrBag{}
	for k, v := range attrs {
		if isPropKey(k, propKeys) {
			propAttrs[k] = v
		} else {
			bagAttrs[k] = v
		}
	}

	// Recursively expand nested x-tags inside the component with this component's props as parent.
	nestedParent := map[string]string{}
	for k, v := range propAttrs {
		nestedParent[k] = v
	}
	for k, v := range parentProps {
		if _, ok := nestedParent[k]; !ok {
			nestedParent[k] = v
		}
	}
	partial, err = e.expandXTags(partial, seen, bags, once, nestedParent)
	if err != nil {
		return "", err
	}
	partial = expandProps(partial)

	for k, v := range slots {
		expanded, err := e.expandXTags(v, seen, bags, once, nestedParent)
		if err != nil {
			return "", err
		}
		slots[k] = expanded
	}

	dictCall := "dict"
	for k, v := range propAttrs {
		dictCall = fmt.Sprintf(`%s %q %q`, dictCall, k, v)
	}
	for k, v := range slots {
		dictCall = fmt.Sprintf(`%s %q (safeStr %q)`, dictCall, k, v)
	}
	// attributes bag as AttrBag via attributesBag helper
	attrPairs := make([]string, 0, len(bagAttrs)*2)
	keys := make([]string, 0, len(bagAttrs))
	for k := range bagAttrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		attrPairs = append(attrPairs, fmt.Sprintf("%q", k), fmt.Sprintf("%q", bagAttrs[k]))
	}
	if len(attrPairs) == 0 {
		dictCall = fmt.Sprintf(`%s "attributes" (attributesBag)`, dictCall)
	} else {
		dictCall = fmt.Sprintf(`%s "attributes" (attributesBag %s)`, dictCall, strings.Join(attrPairs, " "))
	}

	return fmt.Sprintf(`{{ with mergeDict . (%s) }}%s{{ end }}`, dictCall, partial), nil
}

func xComponentViewName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "-", ".")
	if strings.HasPrefix(name, "components.") {
		return name
	}
	return "components." + name
}

func parseHTMLAttributes(attrStr string) map[string]string {
	out := map[string]string{}
	attrStr = strings.TrimSpace(attrStr)
	if attrStr == "" {
		return out
	}
	used := map[string]bool{}
	for _, match := range reAttrPair.FindAllStringSubmatch(attrStr, -1) {
		key := match[1]
		val := match[2]
		if val == "" {
			val = match[3]
		}
		if val == "" {
			val = match[4]
		}
		key = strings.TrimPrefix(key, ":")
		out[key] = val
		used[key] = true
	}
	// Boolean attributes: tokens not already captured as key=value.
	for _, tok := range strings.Fields(attrStr) {
		if strings.Contains(tok, "=") {
			continue
		}
		tok = strings.Trim(tok, `/`)
		if tok == "" || used[tok] {
			continue
		}
		if reBoolAttr.MatchString(tok) {
			out[tok] = tok
		}
	}
	return out
}

func extractAwareKeys(content string) []string {
	match := reAware.FindStringSubmatch(content)
	if len(match) != 2 {
		return nil
	}
	return splitQuotedList(match[1])
}

func extractPropsKeys(content string) map[string]bool {
	match := reProps.FindStringSubmatch(content)
	keys := map[string]bool{}
	if len(match) != 2 {
		return keys
	}
	inner := strings.TrimSpace(match[1])
	inner = strings.TrimPrefix(inner, "[")
	inner = strings.TrimSuffix(inner, "]")
	for _, entry := range splitBladeMapEntries(inner) {
		kv := strings.SplitN(entry, "=>", 2)
		key := strings.Trim(strings.TrimSpace(kv[0]), `'"`)
		if key != "" {
			keys[key] = true
		}
	}
	return keys
}

func isPropKey(key string, propKeys map[string]bool) bool {
	if len(propKeys) == 0 {
		// Without @props, treat non-class/id/style/data/aria as props heuristically:
		// actually Laravel puts undefined in attributes; known @props go to props.
		// With no @props, all become both available as vars AND attributes — use attributes for HTML-ish.
		switch {
		case key == "class", key == "id", key == "style":
			return false
		case strings.HasPrefix(key, "data-"), strings.HasPrefix(key, "aria-"):
			return false
		default:
			return true
		}
	}
	return propKeys[key]
}

func splitQuotedList(expr string) []string {
	parts := splitBladeMapEntries(expr)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(strings.TrimSpace(p), `'"`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func attributesBag(values ...any) AttrBag {
	out := AttrBag{}
	for i := 0; i+1 < len(values); i += 2 {
		out[fmt.Sprint(values[i])] = fmt.Sprint(values[i+1])
	}
	return out
}

func attributesHTML(v any) template.HTMLAttr {
	switch a := v.(type) {
	case AttrBag:
		return template.HTMLAttr(a.String())
	case map[string]string:
		return template.HTMLAttr(AttrBag(a).String())
	case map[string]any:
		bag := AttrBag{}
		for k, val := range a {
			bag[k] = fmt.Sprint(val)
		}
		return template.HTMLAttr(bag.String())
	case template.HTMLAttr:
		return a
	case string:
		return template.HTMLAttr(a)
	default:
		return ""
	}
}
