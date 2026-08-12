package http

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Except returns all inputs except the given keys.
func (r *Request) Except(keys ...string) map[string]string {
	skip := make(map[string]bool, len(keys))
	for _, key := range keys {
		skip[key] = true
	}
	all := r.All()
	out := make(map[string]string, len(all))
	for key, value := range all {
		if skip[key] {
			continue
		}
		out[key] = value
	}
	return out
}

// Has reports whether the input key exists (even if empty).
func (r *Request) Has(key string) bool {
	all := r.All()
	_, ok := all[key]
	return ok
}

// Filled reports whether the input key exists and is non-empty.
func (r *Request) Filled(key string) bool {
	return strings.TrimSpace(r.Input(key)) != ""
}

// Empty reports whether the input key is missing or blank.
func (r *Request) Empty(key string) bool {
	return !r.Filled(key)
}

// Missing reports whether the input key is absent.
func (r *Request) Missing(key string) bool {
	return !r.Has(key)
}

// HasAny reports whether any of the given keys exist.
func (r *Request) HasAny(keys ...string) bool {
	for _, key := range keys {
		if r.Has(key) {
			return true
		}
	}
	return false
}

// HasAll reports whether all of the given keys exist.
func (r *Request) HasAll(keys ...string) bool {
	if len(keys) == 0 {
		return true
	}
	for _, key := range keys {
		if !r.Has(key) {
			return false
		}
	}
	return true
}

// MissingAny reports whether any of the given keys are absent.
func (r *Request) MissingAny(keys ...string) bool {
	for _, key := range keys {
		if r.Missing(key) {
			return true
		}
	}
	return false
}

// FilledAny reports whether any of the given keys are filled.
func (r *Request) FilledAny(keys ...string) bool {
	for _, key := range keys {
		if r.Filled(key) {
			return true
		}
	}
	return false
}

// FilledAll reports whether all of the given keys are filled.
func (r *Request) FilledAll(keys ...string) bool {
	if len(keys) == 0 {
		return true
	}
	for _, key := range keys {
		if !r.Filled(key) {
			return false
		}
	}
	return true
}

// MissingAll reports whether all of the given keys are absent.
func (r *Request) MissingAll(keys ...string) bool {
	if len(keys) == 0 {
		return true
	}
	for _, key := range keys {
		if !r.Missing(key) {
			return false
		}
	}
	return true
}

// Keys returns sorted input keys.
func (r *Request) Keys() []string {
	all := r.All()
	keys := make([]string, 0, len(all))
	for key := range all {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Values returns input values ordered by Keys().
func (r *Request) Values() []string {
	keys := r.Keys()
	all := r.All()
	values := make([]string, len(keys))
	for i, key := range keys {
		values[i] = all[key]
	}
	return values
}

// IsEmpty reports whether the request has no input values.
func (r *Request) IsEmpty() bool {
	return len(r.All()) == 0
}

// IsNotEmpty reports whether the request has at least one input value.
func (r *Request) IsNotEmpty() bool {
	return !r.IsEmpty()
}

// WhenHas runs fn when the key exists (even if empty).
func (r *Request) WhenHas(key string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.Has(key) {
		fn(r)
	}
	return r
}

// WhenFilled runs fn when the key exists and is non-empty.
func (r *Request) WhenFilled(key string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.Filled(key) {
		fn(r)
	}
	return r
}

// WhenMissing runs fn when the key is absent.
func (r *Request) WhenMissing(key string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.Missing(key) {
		fn(r)
	}
	return r
}

// WhenBoolean runs fn when the key parses as a truthy boolean.
func (r *Request) WhenBoolean(key string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.Boolean(key) {
		fn(r)
	}
	return r
}

// WhenTrue runs fn when the key parses as a truthy boolean.
func (r *Request) WhenTrue(key string, fn func(*Request)) *Request {
	return r.WhenBoolean(key, fn)
}

// WhenFalse runs fn when the key does not parse as a truthy boolean.
func (r *Request) WhenFalse(key string, fn func(*Request)) *Request {
	if r != nil && fn != nil && !r.Boolean(key) {
		fn(r)
	}
	return r
}

// WhenEmpty runs fn when the key is missing or blank.
func (r *Request) WhenEmpty(key string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.Empty(key) {
		fn(r)
	}
	return r
}

// WhenHasAny runs fn when any of the given keys exist.
func (r *Request) WhenHasAny(keys []string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.HasAny(keys...) {
		fn(r)
	}
	return r
}

// WhenFilledAny runs fn when any of the given keys are filled.
func (r *Request) WhenFilledAny(keys []string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.FilledAny(keys...) {
		fn(r)
	}
	return r
}

// WhenMissingAny runs fn when any of the given keys are absent.
func (r *Request) WhenMissingAny(keys []string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.MissingAny(keys...) {
		fn(r)
	}
	return r
}

// WhenHasAll runs fn when all of the given keys exist.
func (r *Request) WhenHasAll(keys []string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.HasAll(keys...) {
		fn(r)
	}
	return r
}

// WhenFilledAll runs fn when all of the given keys are filled.
func (r *Request) WhenFilledAll(keys []string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.FilledAll(keys...) {
		fn(r)
	}
	return r
}

// WhenMissingAll runs fn when all of the given keys are absent.
func (r *Request) WhenMissingAll(keys []string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.MissingAll(keys...) {
		fn(r)
	}
	return r
}

// Boolean parses a boolean-ish input value.
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
