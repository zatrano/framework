package http

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (r *Request) IntegerOK(key string) (int, bool) {
	raw := strings.TrimSpace(r.Input(key))
	if raw == "" {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return n, true
}

// FloatOK parses a float and reports success.
func (r *Request) FloatOK(key string) (float64, bool) {
	raw := strings.TrimSpace(r.Input(key))
	if raw == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// BooleanOK reports whether the key is present and a recognized boolean-ish value.
func (r *Request) BooleanOK(key string) (bool, bool) {
	if r.Missing(key) {
		return false, false
	}
	raw := strings.ToLower(strings.TrimSpace(r.Input(key)))
	switch raw {
	case "1", "true", "on", "yes":
		return true, true
	case "0", "false", "off", "no":
		return false, true
	default:
		return false, false
	}
}

// DateOr parses a date input or returns fallback.
func (r *Request) DateOr(key string, fallback time.Time, layout ...string) time.Time {
	if t, ok := r.Date(key, layout...); ok {
		return t
	}
	return fallback
}

// EnumOr returns the enum value or fallback.
func (r *Request) EnumOr(key, fallback string, options ...string) string {
	if value, ok := r.Enum(key, options...); ok {
		return value
	}
	return fallback
}

// Boolean returns a boolean-ish input (missing/unknown is false).
func (r *Request) Boolean(key string) bool {
	switch strings.ToLower(strings.TrimSpace(r.Input(key))) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

// Integer parses an integer input with optional fallback.
func (r *Request) Integer(key string, fallback ...int) int {
	raw := strings.TrimSpace(r.Input(key))
	if raw == "" {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	return n
}

// Float parses a float input with optional fallback.
func (r *Request) Float(key string, fallback ...float64) float64 {
	raw := strings.TrimSpace(r.Input(key))
	if raw == "" {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	return n
}

// String returns a trimmed input string with optional fallback.
func (r *Request) String(key string, fallback ...string) string {
	value := strings.TrimSpace(r.Input(key))
	if value == "" && len(fallback) > 0 {
		return fallback[0]
	}
	return value
}

// Enum returns the input value when it matches one of the options.
func (r *Request) Enum(key string, options ...string) (string, bool) {
	value := r.Input(key)
	for _, opt := range options {
		if value == opt {
			return value, true
		}
	}
	return "", false
}

// Date parses an input value as time.Time using layout (default 2006-01-02).
func (r *Request) Date(key string, layout ...string) (time.Time, bool) {
	raw := strings.TrimSpace(r.Input(key))
	if raw == "" {
		return time.Time{}, false
	}
	format := "2006-01-02"
	if len(layout) > 0 && layout[0] != "" {
		format = layout[0]
	}
	t, err := time.ParseInLocation(format, raw, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// Strings splits a comma-separated input into trimmed non-empty parts.
func (r *Request) Strings(key string) []string {
	raw := strings.TrimSpace(r.Input(key))
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

// Integers parses a comma-separated list of integers (invalid parts skipped).
func (r *Request) Integers(key string) []int {
	parts := r.Strings(key)
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}

// Floats parses a comma-separated list of floats (invalid parts skipped).
func (r *Request) Floats(key string) []float64 {
	parts := r.Strings(key)
	out := make([]float64, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.ParseFloat(part, 64)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}

// Merge merges values into the request input (form + JSON overlay).
func (r *Request) Merge(values map[string]string) {
	if r == nil || len(values) == 0 {
		return
	}
	_ = r.raw.ParseForm()
	if r.raw.Form == nil {
		r.raw.Form = url.Values{}
	}
	data := r.jsonInput()
	for key, value := range values {
		r.raw.Form.Set(key, value)
		if r.raw.PostForm != nil {
			r.raw.PostForm.Set(key, value)
		}
		data[key] = value
	}
}

// MergeIfMissing merges only keys that are currently absent from the request.
func (r *Request) MergeIfMissing(values map[string]string) {
	if r == nil || len(values) == 0 {
		return
	}
	pending := make(map[string]string)
	for key, value := range values {
		if r.Missing(key) {
			pending[key] = value
		}
	}
	r.Merge(pending)
}

// Replace replaces all request inputs with the given values.
func (r *Request) Replace(values map[string]string) {
	if r == nil {
		return
	}
	_ = r.raw.ParseForm()
	r.raw.Form = url.Values{}
	if r.raw.PostForm != nil {
		r.raw.PostForm = url.Values{}
	}
	r.jsonRead = true
	r.jsonData = make(map[string]string, len(values))
	for key, value := range values {
		r.raw.Form.Set(key, value)
		if r.raw.PostForm != nil {
			r.raw.PostForm.Set(key, value)
		}
		r.jsonData[key] = value
	}
}
