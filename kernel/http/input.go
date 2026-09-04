package http

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Input convenience helpers live here, not in request.go.

// Input returns an input value from form, multipart, JSON, or query.
func (r *Request) Input(key string, fallback ...string) string {
	if err := r.raw.ParseForm(); err == nil {
		if value := r.raw.Form.Get(key); value != "" {
			return value
		}
	}
	if err := r.parseMultipart(); err == nil && r.raw.MultipartForm != nil {
		if values := r.raw.MultipartForm.Value[key]; len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}
	if value := r.jsonInput()[key]; value != "" {
		return value
	}
	return r.Query(key, fallback...)
}

// All returns all input values from form and JSON body.
func (r *Request) All() map[string]string {
	_ = r.raw.ParseForm()
	values := make(map[string]string)
	for key, items := range r.raw.Form {
		if len(items) > 0 {
			values[key] = items[0]
		}
	}
	if err := r.parseMultipart(); err == nil && r.raw.MultipartForm != nil {
		for key, items := range r.raw.MultipartForm.Value {
			if _, exists := values[key]; exists {
				continue
			}
			if len(items) > 0 {
				values[key] = items[0]
			}
		}
	}
	for key, value := range r.jsonInput() {
		if _, exists := values[key]; !exists {
			values[key] = value
		}
	}
	return values
}

// TransformInputs mutates form and JSON inputs. When keep is false the key is removed.
func (r *Request) TransformInputs(fn func(key, value string) (string, bool)) {
	if r == nil || r.raw == nil || fn == nil {
		return
	}
	_ = r.raw.ParseForm()
	if r.raw.Form != nil {
		for key, items := range r.raw.Form {
			if len(items) == 0 {
				continue
			}
			next, keep := fn(key, items[0])
			if !keep {
				r.raw.Form.Del(key)
				if r.raw.PostForm != nil {
					r.raw.PostForm.Del(key)
				}
				continue
			}
			r.raw.Form.Set(key, next)
			if r.raw.PostForm != nil {
				r.raw.PostForm.Set(key, next)
			}
		}
	}
	data := r.jsonInput()
	for key, value := range data {
		next, keep := fn(key, value)
		if !keep {
			delete(data, key)
			continue
		}
		data[key] = next
	}
}

func (r *Request) jsonInput() map[string]string {
	if r.jsonRead {
		return r.jsonData
	}
	r.jsonRead = true
	r.jsonData = map[string]string{}
	r.jsonRaw = map[string]any{}
	if r.raw == nil || r.raw.Body == nil || !r.IsJSON() {
		return r.jsonData
	}
	raw, err := r.readBody()
	if err != nil {
		return r.jsonData
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return r.jsonData
	}
	r.jsonRaw = payload
	for key, value := range payload {
		r.jsonData[key] = stringifyJSON(value)
	}
	flattenJSON("", payload, r.jsonData)
	return r.jsonData
}

func stringifyJSON(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return ""
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(raw)
	}
}

// Only returns a subset of input values.
func (r *Request) Only(keys ...string) map[string]string {
	all := r.All()
	selected := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := all[key]; ok {
			selected[key] = value
		}
	}
	return selected
}

// OnlyFilled returns a subset of keys that exist and are non-empty.
func (r *Request) OnlyFilled(keys ...string) map[string]string {
	selected := make(map[string]string)
	for _, key := range keys {
		if r.Filled(key) {
			selected[key] = r.Input(key)
		}
	}
	return selected
}

// ExceptFilled returns all filled inputs except the given keys.
func (r *Request) ExceptFilled(keys ...string) map[string]string {
	skip := make(map[string]bool, len(keys))
	for _, key := range keys {
		skip[key] = true
	}
	all := r.All()
	out := make(map[string]string)
	for key, value := range all {
		if skip[key] || strings.TrimSpace(value) == "" {
			continue
		}
		out[key] = value
	}
	return out
}

// ExceptEmpty returns all non-empty input values.
func (r *Request) ExceptEmpty() map[string]string {
	return r.ExceptFilled()
}

// Exists is an alias for Has.
func (r *Request) Exists(key string) bool {
	return r.Has(key)
}

// AnyFilled is an alias for FilledAny.
func (r *Request) AnyFilled(keys ...string) bool {
	return r.FilledAny(keys...)
}

// EmptyAny reports whether any of the given keys are empty/missing.
func (r *Request) EmptyAny(keys ...string) bool {
	for _, key := range keys {
		if r.Empty(key) {
			return true
		}
	}
	return false
}

// EmptyAll reports whether all of the given keys are empty/missing.
func (r *Request) EmptyAll(keys ...string) bool {
	if len(keys) == 0 {
		return true
	}
	for _, key := range keys {
		if !r.Empty(key) {
			return false
		}
	}
	return true
}

// WhenNotFilled runs fn when the key is missing or blank.
func (r *Request) WhenNotFilled(key string, fn func(*Request)) *Request {
	return r.WhenEmpty(key, fn)
}

// WhenNotEmpty runs fn when the key is filled.
func (r *Request) WhenNotEmpty(key string, fn func(*Request)) *Request {
	return r.WhenFilled(key, fn)
}

// WhenEmptyAny runs fn when any of the given keys are empty/missing.
func (r *Request) WhenEmptyAny(keys []string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.EmptyAny(keys...) {
		fn(r)
	}
	return r
}

// WhenEmptyAll runs fn when all of the given keys are empty/missing.
func (r *Request) WhenEmptyAll(keys []string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.EmptyAll(keys...) {
		fn(r)
	}
	return r
}

// IntegerOK parses an integer and reports success.
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

// Forget removes input keys from form and JSON overlays.
func (r *Request) Forget(keys ...string) {
	if r == nil || r.raw == nil || len(keys) == 0 {
		return
	}
	_ = r.raw.ParseForm()
	data := r.jsonInput()
	for _, key := range keys {
		if r.raw.Form != nil {
			r.raw.Form.Del(key)
		}
		if r.raw.PostForm != nil {
			r.raw.PostForm.Del(key)
		}
		delete(data, key)
	}
}

// Pull returns an input value and removes it from the request.
func (r *Request) Pull(key string, fallback ...string) string {
	value := r.Input(key, fallback...)
	r.Forget(key)
	return value
}

// MergeIfFilled merges only non-empty values.
func (r *Request) MergeIfFilled(values map[string]string) {
	if r == nil || len(values) == 0 {
		return
	}
	pending := make(map[string]string)
	for key, value := range values {
		if strings.TrimSpace(value) != "" {
			pending[key] = value
		}
	}
	r.Merge(pending)
}

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
