package http

import (
	"encoding/json"
	"strings"
)

// Header returns a request header.
func (r *Request) Header(key string, fallback ...string) string {
	value := r.raw.Header.Get(key)
	if value == "" && len(fallback) > 0 {
		return fallback[0]
	}
	return value
}

// BearerToken extracts a bearer token from the Authorization header.
func (r *Request) BearerToken() string {
	header := r.Header("Authorization")
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return ""
}

// WantsJSON reports whether the client explicitly accepts JSON or sent a JSON body.
// Empty Accept and */* are not treated as JSON-specific; q=0 JSON is not a match.
func (r *Request) WantsJSON() bool {
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
	return strings.Contains(r.Header("Content-Type"), "application/json")
}

// JSON decodes the request body into dest. The raw body is not closed.
func (r *Request) JSON(dest any) error {
	raw, err := r.readBody()
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

// HasHeader reports whether a header is present and non-empty.
func (r *Request) HasHeader(key string) bool {
	return r.raw.Header.Get(key) != ""
}

// MissingHeader reports whether a header is absent or empty.
func (r *Request) MissingHeader(key string) bool {
	return !r.HasHeader(key)
}

// WhenHasHeader runs fn when the header is present and non-empty.
func (r *Request) WhenHasHeader(key string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.HasHeader(key) {
		fn(r)
	}
	return r
}

// WhenMissingHeader runs fn when the header is absent or empty.
func (r *Request) WhenMissingHeader(key string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.MissingHeader(key) {
		fn(r)
	}
	return r
}

// HasAnyHeader reports whether any of the given headers are present.
func (r *Request) HasAnyHeader(keys ...string) bool {
	for _, key := range keys {
		if r.HasHeader(key) {
			return true
		}
	}
	return false
}

// HasAllHeaders reports whether all of the given headers are present.
func (r *Request) HasAllHeaders(keys ...string) bool {
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
	for _, key := range keys {
		if r.MissingHeader(key) {
			return true
		}
	}
	return false
}

// MissingAllHeaders reports whether all of the given headers are missing.
func (r *Request) MissingAllHeaders(keys ...string) bool {
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
	if r != nil && fn != nil && r.HasAnyHeader(keys...) {
		fn(r)
	}
	return r
}

// WhenMissingAnyHeader runs fn when any header is missing.
func (r *Request) WhenMissingAnyHeader(keys []string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.MissingAnyHeader(keys...) {
		fn(r)
	}
	return r
}

// HeadersMap returns the first value for each request header.
func (r *Request) HeadersMap() map[string]string {
	if r == nil || r.raw == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(r.raw.Header))
	for key, values := range r.raw.Header {
		if len(values) > 0 {
			out[key] = values[0]
		}
	}
	return out
}

// ContentType returns the request Content-Type header.
func (r *Request) ContentType() string {
	return r.Header("Content-Type")
}

// AcceptsXml reports whether the client accepts XML.
func (r *Request) AcceptsXml() bool {
	return r.Accepts("xml", "application/xml", "text/xml")
}

// PrefersHtml reports whether HTML is preferred among common web types.
func (r *Request) PrefersHtml() bool {
	return r.Prefers("html", "json", "xml") == "html"
}

// IsXmlHttpRequest is an alias for Ajax.
func (r *Request) IsXmlHttpRequest() bool {
	return r.Ajax()
}

// HasAnyCookie reports whether any of the given cookies are present.
func (r *Request) HasAnyCookie(names ...string) bool {
	for _, name := range names {
		if r.HasCookie(name) {
			return true
		}
	}
	return false
}

// HasAllCookies reports whether all of the given cookies are present.
func (r *Request) HasAllCookies(names ...string) bool {
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
	for _, name := range names {
		if r.MissingCookie(name) {
			return true
		}
	}
	return false
}

// MissingAllCookies reports whether all of the given cookies are absent.
func (r *Request) MissingAllCookies(names ...string) bool {
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
	if r != nil && fn != nil && r.HasAnyCookie(names...) {
		fn(r)
	}
	return r
}

// WhenMissingAnyCookie runs fn when any cookie is absent.
func (r *Request) WhenMissingAnyCookie(names []string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.MissingAnyCookie(names...) {
		fn(r)
	}
	return r
}

// CookieMap returns request cookies as name→value.
func (r *Request) CookieMap() map[string]string {
	if r == nil || r.raw == nil {
		return map[string]string{}
	}
	out := make(map[string]string)
	for _, c := range r.raw.Cookies() {
		out[c.Name] = c.Value
	}
	return out
}
