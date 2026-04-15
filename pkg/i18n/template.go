package i18n

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// mapDotField matches a single-segment field in text/template form: {{.Name}} (not {{.a.b}}).
var mapDotField = regexp.MustCompile(`\{\{\s*\.([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)

// mapDotFieldsToIndex rewrites {{.Key}} to {{index . "Key"}} so map data works with Go's text/template.
func mapDotFieldsToIndex(s string) string {
	return mapDotField.ReplaceAllStringFunc(s, func(m string) string {
		sub := mapDotField.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		return fmt.Sprintf(`{{index . %q}}`, sub[1])
	})
}

// Format resolves the message for locale/key, then executes it as a text/template with data.
// Use placeholders like {{.Name}} in JSON strings. For map[string]any or map[string]string, simple
// {{.Name}} segments are rewritten automatically; structs and pointers use normal template rules.
// Nil data returns the raw translation string. Empty data values are fine.
func (b *Bundle) Format(locale, key string, data any) (string, error) {
	raw := b.T(locale, key)
	return formatRaw(raw, data)
}

func formatRaw(raw string, data any) (string, error) {
	if data == nil {
		return raw, nil
	}
	if strings.TrimSpace(raw) == "" {
		return raw, nil
	}

	s := raw
	switch data.(type) {
	case map[string]any, map[string]string:
		s = mapDotFieldsToIndex(s)
	}

	tmpl, err := template.New("i18n").Option("missingkey=default").Parse(s)
	if err != nil {
		return "", fmt.Errorf("i18n template parse: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("i18n template execute: %w", err)
	}
	return buf.String(), nil
}

