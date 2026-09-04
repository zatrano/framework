package html

import (
	"html"
	"regexp"
	"strings"
)

var (
	tagRE     = regexp.MustCompile(`(?is)<[^>]*>`)
	scriptRE  = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRE   = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	onEventRE = regexp.MustCompile(`(?is)\s+on\w+\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
	jsHrefRE  = regexp.MustCompile(`(?is)\s(href|src)\s*=\s*("|')\s*javascript:[^"']*("|')`)
)

// Escape escapes HTML special characters.
func Escape(s string) string {
	return html.EscapeString(s)
}

// Unescape unescapes HTML entities.
func Unescape(s string) string {
	return html.UnescapeString(s)
}

// StripTags removes all HTML tags.
func StripTags(s string) string {
	s = scriptRE.ReplaceAllString(s, "")
	s = styleRE.ReplaceAllString(s, "")
	return strings.TrimSpace(tagRE.ReplaceAllString(s, ""))
}

// Sanitize removes scripts/styles/event handlers and optionally strips all tags.
func Sanitize(s string, allowTags ...string) string {
	s = scriptRE.ReplaceAllString(s, "")
	s = styleRE.ReplaceAllString(s, "")
	s = onEventRE.ReplaceAllString(s, "")
	s = jsHrefRE.ReplaceAllString(s, "")
	if len(allowTags) == 0 {
		return StripTags(s)
	}
	allowed := map[string]struct{}{}
	for _, t := range allowTags {
		allowed[strings.ToLower(t)] = struct{}{}
	}
	return tagRE.ReplaceAllStringFunc(s, func(tag string) string {
		name := tagName(tag)
		if _, ok := allowed[name]; ok {
			return tag
		}
		return ""
	})
}

func tagName(tag string) string {
	tag = strings.TrimSpace(tag)
	tag = strings.TrimPrefix(tag, "<")
	tag = strings.TrimPrefix(tag, "/")
	tag = strings.TrimSuffix(tag, ">")
	fields := strings.Fields(tag)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(fields[0])
}
