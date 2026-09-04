package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	stdhttp "net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/zatrano/framework/kernel/cookie"
)

// Request is the HTTP primitive: method, URL, headers, body, route, cookies,
// session, and attributes. Input convenience helpers belong in input.go.

// ErrBodyTooLarge is returned when JSON/Body reads exceed MaxBodyBytes.
var ErrBodyTooLarge = errors.New("http: request body too large")

// Request wraps the standard HTTP request with helpers.
type Request struct {
	raw          *stdhttp.Request
	route        map[string]string
	attrs        map[string]any
	session      SessionStore
	cookies      *cookie.Jar
	jsonData     map[string]string
	jsonRaw      map[string]any
	jsonRead     bool
	bodyCached   []byte
	bodyErr      error
	bodyRead     bool
	maxBodyBytes int64
}

// SessionStore is the session contract used by requests.
type SessionStore interface {
	Get(key string, fallback ...any) any
	Put(key string, value any)
	Flash(key string, value any)
	Pull(key string, fallback ...any) any
	Forget(key string)
	Regenerate() error
	ID() string
}

// NewRequest creates a ZATRANO request.
func NewRequest(r *stdhttp.Request) *Request {
	return &Request{
		raw:     r,
		route:   make(map[string]string),
		attrs:   make(map[string]any),
		cookies: cookie.NewJar(),
	}
}

// Raw returns the underlying net/http request.
func (r *Request) Raw() *stdhttp.Request {
	return r.raw
}

// Method returns the HTTP method.
func (r *Request) Method() string {
	return r.raw.Method
}

// Path returns the request path.
func (r *Request) Path() string {
	return r.raw.URL.Path
}

// URL returns the full request URL string.
func (r *Request) URL() string {
	return r.raw.URL.String()
}

// Query returns a query parameter.
func (r *Request) Query(key string, fallback ...string) string {
	value := r.raw.URL.Query().Get(key)
	if value == "" && len(fallback) > 0 {
		return fallback[0]
	}
	return value
}

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

// IP attempts to determine the client IP address.
// Prefer a value set by trusted proxy middleware; otherwise use the remote address.
func (r *Request) IP() string {
	if v, ok := r.Get("_client_ip").(string); ok && v != "" {
		return v
	}
	return r.RemoteIP()
}

// RemoteIP returns the direct connection IP (ignores forwarding headers).
func (r *Request) RemoteIP() string {
	if r.raw == nil {
		return ""
	}
	host := r.raw.RemoteAddr
	if strings.HasPrefix(host, "[") {
		if end := strings.Index(host, "]"); end != -1 {
			return host[1:end]
		}
	}
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}

// WantsJSON reports whether the client expects JSON.
func (r *Request) WantsJSON() bool {
	accept := r.Header("Accept")
	return strings.Contains(accept, "application/json") || r.IsJSON()
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

// Body reads the raw request body. The raw body is not closed.
func (r *Request) Body() ([]byte, error) {
	return r.readBody()
}

// SetMaxBodyBytes overrides the default JSON/raw body limit for this request.
func (r *Request) SetMaxBodyBytes(n int64) {
	if r == nil {
		return
	}
	r.maxBodyBytes = n
}

func (r *Request) bodyLimit() int64 {
	if r != nil && r.maxBodyBytes > 0 {
		return r.maxBodyBytes
	}
	return MaxBodyBytes()
}

func (r *Request) readBody() ([]byte, error) {
	if r == nil || r.raw == nil {
		return nil, nil
	}
	if r.bodyRead {
		return r.bodyCached, r.bodyErr
	}
	r.bodyRead = true
	if r.raw.Body == nil {
		return nil, nil
	}
	limit := r.bodyLimit()
	raw, err := io.ReadAll(io.LimitReader(r.raw.Body, limit+1))
	if err != nil {
		r.bodyErr = err
		return nil, err
	}
	if int64(len(raw)) > limit {
		r.bodyErr = ErrBodyTooLarge
		return nil, ErrBodyTooLarge
	}
	r.bodyCached = raw
	r.raw.Body = io.NopCloser(bytes.NewReader(raw))
	return raw, nil
}

// Route returns a route parameter.
func (r *Request) Route(key string, fallback ...string) string {
	if value, ok := r.route[key]; ok {
		return value
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

// RouteInt returns a route parameter as int.
func (r *Request) RouteInt(key string, fallback ...int) int {
	value := r.Route(key)
	if value == "" {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	return parsed
}

// SetRouteParams sets matched route parameters.
func (r *Request) SetRouteParams(params map[string]string) {
	r.route = params
}

// RouteParams returns all matched route parameters.
func (r *Request) RouteParams() map[string]string {
	out := make(map[string]string, len(r.route))
	for key, value := range r.route {
		out[key] = value
	}
	return out
}

// Set sets a request attribute.
func (r *Request) Set(key string, value any) {
	r.attrs[key] = value
}

// Get returns a request attribute.
func (r *Request) Get(key string) any {
	return r.attrs[key]
}

// SetSession attaches a session store.
func (r *Request) SetSession(store SessionStore) {
	r.session = store
}

// Session returns the session store.
func (r *Request) Session() SessionStore {
	return r.session
}

// Cookie returns a cookie value.
func (r *Request) Cookie(name string, fallback ...string) string {
	c, err := r.raw.Cookie(name)
	if err != nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	return c.Value
}

// HasCookie reports whether a cookie is present.
func (r *Request) HasCookie(name string) bool {
	if r == nil || r.raw == nil {
		return false
	}
	_, err := r.raw.Cookie(name)
	return err == nil
}

// MissingCookie reports whether a cookie is absent.
func (r *Request) MissingCookie(name string) bool {
	return !r.HasCookie(name)
}

// WhenHasCookie runs fn when the cookie is present.
func (r *Request) WhenHasCookie(name string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.HasCookie(name) {
		fn(r)
	}
	return r
}

// WhenMissingCookie runs fn when the cookie is absent.
func (r *Request) WhenMissingCookie(name string, fn func(*Request)) *Request {
	if r != nil && fn != nil && r.MissingCookie(name) {
		fn(r)
	}
	return r
}

// Cookies returns the response cookie jar for this request.
func (r *Request) Cookies() *cookie.Jar {
	if r.cookies == nil {
		r.cookies = cookie.NewJar()
	}
	return r.cookies
}

// HasHeader reports whether a header exists.
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

// IsGet reports whether the method is GET.
func (r *Request) IsGet() bool { return r.IsMethod("GET") }

// IsPost reports whether the method is POST.
func (r *Request) IsPost() bool { return r.IsMethod("POST") }

// IsPut reports whether the method is PUT.
func (r *Request) IsPut() bool { return r.IsMethod("PUT") }

// IsPatch reports whether the method is PATCH.
func (r *Request) IsPatch() bool { return r.IsMethod("PATCH") }

// IsDelete reports whether the method is DELETE.
func (r *Request) IsDelete() bool { return r.IsMethod("DELETE") }

// IsHead reports whether the method is HEAD.
func (r *Request) IsHead() bool { return r.IsMethod("HEAD") }

// IsOptions reports whether the method is OPTIONS.
func (r *Request) IsOptions() bool { return r.IsMethod("OPTIONS") }

// IsMethodSafe reports whether the method is GET or HEAD.
func (r *Request) IsMethodSafe() bool {
	return r.IsMethod("GET", "HEAD")
}

// IsMethodIdempotent reports whether the method is GET, HEAD, PUT, DELETE, OPTIONS, or TRACE.
func (r *Request) IsMethodIdempotent() bool {
	return r.IsMethod("GET", "HEAD", "PUT", "DELETE", "OPTIONS", "TRACE")
}

// HasQuery reports whether a query parameter exists (even if empty).
func (r *Request) HasQuery(key string) bool {
	if r == nil || r.raw == nil || r.raw.URL == nil {
		return false
	}
	_, ok := r.raw.URL.Query()[key]
	return ok
}

// QueryInt parses a query parameter as int with optional fallback.
func (r *Request) QueryInt(key string, fallback ...int) int {
	raw := strings.TrimSpace(r.Query(key))
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

// QueryFloat parses a query parameter as float64 with optional fallback.
func (r *Request) QueryFloat(key string, fallback ...float64) float64 {
	raw := strings.TrimSpace(r.Query(key))
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

// QueryBool parses a query parameter as boolean-ish.
func (r *Request) QueryBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(r.Query(key))) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

// Port returns the request host port when present.
func (r *Request) Port() string {
	host := r.Host()
	if host == "" {
		return ""
	}
	_, port, err := net.SplitHostPort(host)
	if err != nil {
		if r.Secure() {
			return "443"
		}
		return "80"
	}
	return port
}

// HttpHost returns the host header value (may include port).
func (r *Request) HttpHost() string {
	return r.Host()
}

// DecodedPath returns the URL-decoded request path.
func (r *Request) DecodedPath() string {
	path := r.Path()
	decoded, err := url.PathUnescape(path)
	if err != nil {
		return path
	}
	return decoded
}

// QueryString returns the raw URL query string without leading ?.
func (r *Request) QueryString() string {
	if r == nil || r.raw == nil || r.raw.URL == nil {
		return ""
	}
	return r.raw.URL.RawQuery
}

// RequestURI returns path + query (RequestURI).
func (r *Request) RequestURI() string {
	if r == nil || r.raw == nil || r.raw.URL == nil {
		return "/"
	}
	uri := r.raw.URL.RequestURI()
	if uri == "" {
		return "/"
	}
	return uri
}

// FullUrlWithQuery returns FullURL with additional/overridden query parameters.
func (r *Request) FullUrlWithQuery(extra map[string]string) string {
	values := url.Values{}
	for key, items := range r.QueryAll() {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	for key, value := range extra {
		values.Set(key, value)
	}
	path := r.Path()
	if path == "" {
		path = "/"
	}
	encoded := values.Encode()
	if encoded == "" {
		return r.Root() + path
	}
	return r.Root() + path + "?" + encoded
}

// FullUrlWithoutQuery returns FullURL without the given query keys.
func (r *Request) FullUrlWithoutQuery(keys ...string) string {
	skip := make(map[string]bool, len(keys))
	for _, key := range keys {
		skip[key] = true
	}
	values := url.Values{}
	for key, items := range r.QueryAll() {
		if skip[key] {
			continue
		}
		for _, item := range items {
			values.Add(key, item)
		}
	}
	path := r.Path()
	if path == "" {
		path = "/"
	}
	encoded := values.Encode()
	if encoded == "" {
		return r.Root() + path
	}
	return r.Root() + path + "?" + encoded
}

// Ips returns client IP candidates (trusted client IP first, then remote).
func (r *Request) Ips() []string {
	seen := map[string]bool{}
	out := make([]string, 0, 2)
	add := func(ip string) {
		ip = strings.TrimSpace(ip)
		if ip == "" || seen[ip] {
			return
		}
		seen[ip] = true
		out = append(out, ip)
	}
	add(r.IP())
	add(r.RemoteIP())
	return out
}

// IsSecure is an alias for Secure.
func (r *Request) IsSecure() bool {
	return r.Secure()
}

// HasFileAny reports whether any of the given file keys are present.
func (r *Request) HasFileAny(keys ...string) bool {
	for _, key := range keys {
		if r.HasFile(key) {
			return true
		}
	}
	return false
}
