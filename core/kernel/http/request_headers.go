package http

import (
	"errors"
	"net/textproto"
	"strings"
)

// Header returns a request header.
func (r *Request) Header(key string, fallback ...string) string {
	if r == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	if r.headerOverlay == nil && len(r.headerDeleted) == 0 {
		if v := r.carrier.header(key); v != "" {
			return v
		}
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	canon := textproto.CanonicalMIMEHeaderKey(key)
	if r.headerDeleted[canon] {
		if len(fallback) > 0 {
			return fallback[0]
		}
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
	if v := r.carrier.header(key); v != "" {
		return v
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

// HeaderValues returns every value of a request header, one entry per header
// line, in the order the client sent them. An overlay replaces the parsed
// values.
func (r *Request) HeaderValues(key string) []string {
	if r == nil {
		return nil
	}
	canon := textproto.CanonicalMIMEHeaderKey(key)
	if r.headerDeleted[canon] {
		return nil
	}
	if r.headerOverlay != nil {
		if vals, ok := r.headerOverlay[canon]; ok {
			return append([]string(nil), vals...)
		}
	}
	return r.carrier.headerValues(key)
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
	return unmarshalJSON(raw, dest)
}

// BindJSON decodes the request body into dest. The body is the request's own
// copy, so a type with UnmarshalJSON may keep the input slice, but it must not
// modify it: Body returns the same bytes.
func (r *Request) BindJSON(dest any) error {
	if r == nil {
		return errors.New("http: nil request")
	}
	if dest == nil {
		return errors.New("http: nil json dest")
	}
	limit := r.bodyLimit()
	if r.bodyRead {
		if r.bodyErr != nil {
			return r.bodyErr
		}
		if int64(len(r.bodyCached)) > limit {
			return ErrBodyTooLarge
		}
		return unmarshalJSON(r.bodyCached, dest)
	}
	if r.bodyOverrideSet {
		if int64(len(r.bodyOverride)) > limit {
			r.bodyRead = true
			r.bodyErr = ErrBodyTooLarge
			return ErrBodyTooLarge
		}
		return unmarshalJSON(r.bodyOverride, dest)
	}
	if raw := r.carrier.body; len(raw) > 0 {
		if int64(len(raw)) > limit {
			r.bodyRead = true
			r.bodyErr = ErrBodyTooLarge
			return ErrBodyTooLarge
		}
		return unmarshalJSON(raw, dest)
	}
	if r.bodyReader != nil {
		raw, err := r.readBody()
		if err != nil {
			return err
		}
		return unmarshalJSON(raw, dest)
	}
	return unmarshalJSON(nil, dest)
}

// HasHeader reports whether a header is present and non-empty.
func (r *Request) HasHeader(key string) bool {
	return r.Header(key) != ""
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
	if r == nil {
		return map[string]string{}
	}
	out := make(map[string]string)
	r.carrier.visitHeaders(func(k, v string) {
		canon := textproto.CanonicalMIMEHeaderKey(k)
		if r.headerDeleted[canon] {
			return
		}
		if _, exists := out[canon]; !exists {
			out[canon] = v
		}
	})
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
	if r == nil {
		return map[string]string{}
	}
	out := make(map[string]string)
	r.carrier.visitCookies(func(name, val string, _ bool) bool {
		out[name] = val
		return true
	})
	for name, value := range r.cookieOverlay {
		out[name] = value
	}
	return out
}
