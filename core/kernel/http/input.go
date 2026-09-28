package http

import (
	"sort"
	"strings"
)

// Input convenience helpers live here, not in request.go.

// Input returns an input value from form, multipart, JSON, or query.
func (r *Request) Input(key string, fallback ...string) string {
	r.applyPendingInputTransforms()
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
	r.applyPendingInputTransforms()
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

// TransformInputs queues a mutation of form and JSON overlay values.
// Transforms run on the first Input/All access (and Merge/Replace/Forget).
// JSON() and Body() read the raw body and do not apply transforms.
// Raw().Form stays untransformed until those accessors run.
func (r *Request) TransformInputs(fn func(key, value string) (string, bool)) {
	if r == nil || r.raw == nil || fn == nil {
		return
	}
	r.inputTransforms = append(r.inputTransforms, fn)
	if r.inputTransformed {
		r.applyOneInputTransform(fn)
	}
}

func (r *Request) applyPendingInputTransforms() {
	if r == nil || r.inputTransformed {
		return
	}
	r.inputTransformed = true
	if r.raw == nil || len(r.inputTransforms) == 0 {
		return
	}
	_ = r.raw.ParseForm()
	r.ensureJSONParsed()
	for _, fn := range r.inputTransforms {
		r.applyOneInputTransform(fn)
	}
}

func (r *Request) applyOneInputTransform(fn func(key, value string) (string, bool)) {
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
	r.ensureJSONParsed()
	data := r.jsonData
	for key, value := range data {
		next, keep := fn(key, value)
		if !keep {
			delete(data, key)
			continue
		}
		data[key] = next
	}
}

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

// Forget removes input keys from form and JSON overlays.
func (r *Request) Forget(keys ...string) {
	if r == nil || r.raw == nil || len(keys) == 0 {
		return
	}
	r.applyPendingInputTransforms()
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
