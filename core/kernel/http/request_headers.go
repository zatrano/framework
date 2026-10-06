package http

import (
	"encoding/json"
	"net/textproto"
	"strings"
)

// Header returns a request header. The first call builds a canonical map;
// later calls read that map. HeaderValue does not build it.
func (r *Request) Header(key string, fallback ...string) string {
	r.poisonCheck()
	if r == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	if r.builtHeaders == nil {
		r.builtHeaders = make(map[string]string)
	}
	canon := textproto.CanonicalMIMEHeaderKey(key)
	if v, ok := r.builtHeaders[canon]; ok {
		if v == "" && len(fallback) > 0 {
			return fallback[0]
		}
		return v
	}
	v := r.headerLookup(canon, key)
	r.builtHeaders[canon] = v
	if v == "" && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// HeaderCacheBuilt reports whether Header has materialized its map.
func (r *Request) HeaderCacheBuilt() bool {
	r.poisonCheck()
	return r != nil && r.builtHeaders != nil
}

func (r *Request) headerLookup(canon, key string) string {
	r.poisonCheck()
	if r.headerDeleted[canon] {
		return ""
	}
	if r.headerOverlay != nil {
		if vals, ok := r.headerOverlay[canon]; ok {
			if len(vals) > 0 {
				return vals[0]
			}
			return ""
		}
	}
	if r.ctx != nil {
		if v := r.ctx.Header(key); len(v) > 0 {
			return string(v)
		}
	}
	return ""
}

// HeaderValue returns one header without building Header's map.
// A missing header allocates nothing. The bool is false when the header is absent.
func (r *Request) HeaderValue(name string) (string, bool) {
	r.poisonCheck()
	b := r.headerBytes(name)
	if len(b) == 0 {
		return "", false
	}
	return string(b), true
}

// headerBytes returns the carrier's header bytes. The slice is borrowed from
// the request buffer and is valid only until the handler returns. An overlay
// value is an owned copy. Do not retain the slice.
func (r *Request) headerBytes(name string) []byte {
	r.poisonCheck()
	if r == nil || name == "" {
		return nil
	}
	if r.headerDeleted != nil {
		canon := textproto.CanonicalMIMEHeaderKey(name)
		if r.headerDeleted[canon] {
			return nil
		}
	}
	if r.headerOverlay != nil {
		canon := textproto.CanonicalMIMEHeaderKey(name)
		if vals, ok := r.headerOverlay[canon]; ok {
			if len(vals) == 0 || vals[0] == "" {
				return nil
			}
			return []byte(vals[0])
		}
	}
	if r.ctx != nil {
		return r.ctx.Header(name)
	}
	return nil
}

// HeaderValues returns every value of a request header.
// An overlay stores each value; the carrier stores one.
// The rawhttp path keeps the last Origin when the request repeats it.
func (r *Request) HeaderValues(key string) []string {
	r.poisonCheck()
	if r == nil {
		return nil
	}
	if r.headerOverlay == nil && r.headerDeleted == nil {
		b := r.headerBytes(key)
		if len(b) == 0 {
			return nil
		}
		return []string{string(b)}
	}
	canon := textproto.CanonicalMIMEHeaderKey(key)
	if r.headerDeleted[canon] {
		return nil
	}
	if r.headerOverlay != nil {
		if vals, ok := r.headerOverlay[canon]; ok {
			return vals
		}
	}
	if r.ctx != nil {
		if v := r.ctx.Header(key); len(v) > 0 {
			return []string{string(v)}
		}
	}
	return nil
}

// BearerToken extracts a bearer token from the Authorization header.
func (r *Request) BearerToken() string {
	r.poisonCheck()
	header := r.Header("Authorization")
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return ""
}

// WantsJSON reports whether the client explicitly accepts JSON or sent a JSON body.
// Empty Accept and */* are not treated as JSON-specific; q=0 JSON is not a match.
func (r *Request) WantsJSON() bool {
	r.poisonCheck()
	if r.IsJSON() {
		return true
	}
	for _, media := range r.acceptableTypes() {
		if media == "*/*" {
			continue
		}
		if typeMatchesAccept("json", media) {
			return true
		}
	}
	return false
}

// IsJSON reports whether the request content type is JSON.
func (r *Request) IsJSON() bool {
	r.poisonCheck()
	return strings.Contains(r.Header("Content-Type"), "application/json")
}

// JSON decodes the request body into dest. The raw body is not closed.
func (r *Request) JSON(dest any) error {
	r.poisonCheck()
	raw, err := r.readBody()
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

// HasHeader reports whether a header is present and non-empty.
func (r *Request) HasHeader(key string) bool {
	r.poisonCheck()
	return r.Header(key) != ""
}

// MissingHeader reports whether a header is absent or empty.
func (r *Request) MissingHeader(key string) bool {
	r.poisonCheck()
	return !r.HasHeader(key)
}

// WhenHasHeader runs fn when the header is present and non-empty.
func (r *Request) WhenHasHeader(key string, fn func(*Request)) *Request {
	r.poisonCheck()
	if r != nil && fn != nil && r.HasHeader(key) {
		fn(r)
	}
	return r
}

// WhenMissingHeader runs fn when the header is absent or empty.
func (r *Request) WhenMissingHeader(key string, fn func(*Request)) *Request {
	r.poisonCheck()
	if r != nil && fn != nil && r.MissingHeader(key) {
		fn(r)
	}
	return r
}

// HasAnyHeader reports whether any of the given headers are present.
func (r *Request) HasAnyHeader(keys ...string) bool {
	r.poisonCheck()
	for _, key := range keys {
		if r.HasHeader(key) {
			return true
		}
	}
	return false
}

// HasAllHeaders reports whether all of the given headers are present.
func (r *Request) HasAllHeaders(keys ...string) bool {
	r.poisonCheck()
	if len(keys) == 0 {
		return true
	}
	for _, key := range keys {
		if !r.HasHeader(key) {
			return false
		}
	}
	return true
}

// MissingAnyHeader reports whether any of the given headers are missing.
func (r *Request) MissingAnyHeader(keys ...string) bool {
	r.poisonCheck()
	for _, key := range keys {
		if r.MissingHeader(key) {
			return true
		}
	}
	return false
}

// MissingAllHeaders reports whether all of the given headers are missing.
func (r *Request) MissingAllHeaders(keys ...string) bool {
	r.poisonCheck()
	if len(keys) == 0 {
		return true
	}
	for _, key := range keys {
		if !r.MissingHeader(key) {
			return false
		}
	}
	return true
}

// WhenHasAnyHeader runs fn when any header is present.
func (r *Request) WhenHasAnyHeader(keys []string, fn func(*Request)) *Request {
	r.poisonCheck()
	if r != nil && fn != nil && r.HasAnyHeader(keys...) {
		fn(r)
	}
	return r
}

// WhenMissingAnyHeader runs fn when any header is missing.
func (r *Request) WhenMissingAnyHeader(keys []string, fn func(*Request)) *Request {
	r.poisonCheck()
	if r != nil && fn != nil && r.MissingAnyHeader(keys...) {
		fn(r)
	}
	return r
}

// HeadersMap returns the first value for each request header.
func (r *Request) HeadersMap() map[string]string {
	r.poisonCheck()
	if r == nil {
		return map[string]string{}
	}
	out := make(map[string]string)
	if r.ctx != nil {
		r.ctx.VisitHeader(func(k, v []byte) {
			key := string(k)
			canon := textproto.CanonicalMIMEHeaderKey(key)
			if r.headerDeleted[canon] {
				return
			}
			if _, exists := out[canon]; !exists {
				out[canon] = string(v)
			}
		})
	}
	for key, values := range r.headerOverlay {
		if r.headerDeleted[key] {
			continue
		}
		if len(values) > 0 {
			out[key] = values[0]
		}
	}
	return out
}

// ContentType returns the request Content-Type header.
func (r *Request) ContentType() string {
	r.poisonCheck()
	return r.Header("Content-Type")
}

// AcceptsXml reports whether the client accepts XML.
func (r *Request) AcceptsXml() bool {
	r.poisonCheck()
	return r.Accepts("xml", "application/xml", "text/xml")
}

// PrefersHtml reports whether HTML is preferred among common web types.
func (r *Request) PrefersHtml() bool {
	r.poisonCheck()
	return r.Prefers("html", "json", "xml") == "html"
}

// IsXmlHttpRequest is an alias for Ajax.
func (r *Request) IsXmlHttpRequest() bool {
	r.poisonCheck()
	return r.Ajax()
}

// HasAnyCookie reports whether any of the given cookies are present.
func (r *Request) HasAnyCookie(names ...string) bool {
	r.poisonCheck()
	for _, name := range names {
		if r.HasCookie(name) {
			return true
		}
	}
	return false
}

// HasAllCookies reports whether all of the given cookies are present.
func (r *Request) HasAllCookies(names ...string) bool {
	r.poisonCheck()
	if len(names) == 0 {
		return true
	}
	for _, name := range names {
		if !r.HasCookie(name) {
			return false
		}
	}
	return true
}

// MissingAnyCookie reports whether any of the given cookies are absent.
func (r *Request) MissingAnyCookie(names ...string) bool {
	r.poisonCheck()
	for _, name := range names {
		if r.MissingCookie(name) {
			return true
		}
	}
	return false
}

// MissingAllCookies reports whether all of the given cookies are absent.
func (r *Request) MissingAllCookies(names ...string) bool {
	r.poisonCheck()
	if len(names) == 0 {
		return true
	}
	for _, name := range names {
		if !r.MissingCookie(name) {
			return false
		}
	}
	return true
}

// WhenHasAnyCookie runs fn when any cookie is present.
func (r *Request) WhenHasAnyCookie(names []string, fn func(*Request)) *Request {
	r.poisonCheck()
	if r != nil && fn != nil && r.HasAnyCookie(names...) {
		fn(r)
	}
	return r
}

// WhenMissingAnyCookie runs fn when any cookie is absent.
func (r *Request) WhenMissingAnyCookie(names []string, fn func(*Request)) *Request {
	r.poisonCheck()
	if r != nil && fn != nil && r.MissingAnyCookie(names...) {
		fn(r)
	}
	return r
}

// CookieMap returns request cookies as name→value.
func (r *Request) CookieMap() map[string]string {
	r.poisonCheck()
	if r == nil {
		return map[string]string{}
	}
	out := make(map[string]string)
	if r.ctx != nil {
		r.ctx.VisitCookie(func(name, val []byte) {
			out[string(name)] = string(val)
		})
	}
	for name, value := range r.cookieOverlay {
		out[name] = value
	}
	return out
}
