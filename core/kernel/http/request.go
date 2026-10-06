package http

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net"
	stdhttp "net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/zatrano/framework/v3/core/kernel/cookie"

	"github.com/zatrano/rawhttp"
)

// Request is the HTTP primitive: method, URL, headers, body, route, cookies,
// session, and attributes. Input convenience helpers belong in input.go.

// ErrBodyTooLarge is returned when JSON/Body reads exceed MaxBodyBytes.
var ErrBodyTooLarge = errors.New("http: request body too large")

var requestPool = sync.Pool{New: func() any { return new(Request) }}

// requestPoolOn is read on every request. The default is off (a new Request
// per request, the same as v3.0.1). HTTP_POOL_REQUESTS turns it on at boot.
var requestPoolOn atomic.Bool

// Request wraps rawhttp.Ctx with optional mutable overlays for middleware/tests.
type Request struct {
	ctx *rawhttp.Ctx

	method     string
	methodSet  bool
	path       string
	pathSet    bool
	query      string
	querySet   bool
	host       string
	remoteAddr string
	secure     bool
	secureSet  bool

	headerOverlay stdhttp.Header
	headerDeleted map[string]bool
	// builtHeaders is filled by Header. HeaderValue does not create it.
	builtHeaders  map[string]string
	cookieOverlay map[string]string

	stdCtx context.Context

	form          url.Values
	postForm      url.Values
	formParsed    bool
	multipartForm *multipart.Form

	route            map[string]string
	attrs            map[string]any
	cookies          *cookie.Jar
	jsonData         map[string]string
	jsonRaw          map[string]any
	jsonRead         bool
	inputTransforms  []func(key, value string) (string, bool)
	inputTransformed bool
	bodyCached       []byte
	bodyErr          error
	bodyRead         bool
	bodyOverride     []byte
	bodyOverrideSet  bool
	bodyReader       io.ReadCloser
	maxBodyBytes     int64
	// freed is set only on poison builds, after the handler returns.
	freed bool
}

// SessionStore is an optional request capability (flash/csrf bag).
// Kernel HTTP does not import the session package; the implementation is
// attached as a request attribute.
type SessionStore interface {
	Get(key string, fallback ...any) any
	Put(key string, value any)
	Flash(key string, value any)
	Pull(key string, fallback ...any) any
	Forget(key string)
	Regenerate() error
	ID() string
}

// ConfigureRequestPool turns Request reuse on or off. The application calls
// this once at boot from HTTP_POOL_REQUESTS. The default is off: each
// request gets a new object, and that object stays readable after the
// handler returns. When the pool is on, Application.Handle returns the
// object after Commit and a later request may reuse it. Do not keep the
// pointer. Copy what you need with Clone before returning.
func ConfigureRequestPool(on bool) { requestPoolOn.Store(on) }

// RequestPoolEnabled reports whether Request objects are reused.
func RequestPoolEnabled() bool { return requestPoolOn.Load() }

// NewRequest creates a ZATRANO request from a rawhttp context.
// Pooling is off unless ConfigureRequestPool was called.
// Path, QueryString, and HeaderValue strings are copies. headerBytes and
// PathBytes alias the request buffer and are invalid after the handler returns.
func NewRequest(ctx *rawhttp.Ctx) *Request {
	if requestPoolOn.Load() {
		r := requestPool.Get().(*Request)
		r.reset()
		r.ctx = ctx
		r.stdCtx = context.Background()
		return r
	}
	return &Request{ctx: ctx, stdCtx: context.Background()}
}

// ReleaseRequest ends the request's handler lifetime.
// With the pool off this only marks the object freed on poison builds; the
// fields stay so a later read still sees this request.
// With the pool on the object is cleared, marked freed until the next
// checkout, and returned to the pool.
func ReleaseRequest(r *Request) {
	if r == nil {
		return
	}
	if requestPoolOn.Load() {
		r.markFreed()
		kept := r.freed
		r.reset()
		r.freed = kept
		requestPool.Put(r)
		return
	}
	r.markFreed()
}

func (r *Request) reset() {
	*r = Request{}
}

// RequestFromHTTP copies a net/http request into overlays for unit tests.
// It does not keep a synthetic *http.Request adapter.
func RequestFromHTTP(sr *stdhttp.Request) *Request {
	if sr == nil {
		return NewRequest(nil)
	}
	method := sr.Method
	if method == "" {
		method = stdhttp.MethodGet
	}
	path := "/"
	query := ""
	host := sr.Host
	if sr.URL != nil {
		if sr.URL.Path != "" {
			path = sr.URL.Path
		} else if sr.URL.Opaque == "" {
			path = "/"
		}
		query = sr.URL.RawQuery
		if host == "" {
			host = sr.URL.Host
		}
	}
	ctx := &rawhttp.Ctx{
		Method: []byte(method),
		Path:   []byte(path),
		Query:  []byte(query),
	}
	req := NewRequest(ctx)
	req.host = host
	req.remoteAddr = sr.RemoteAddr
	if sr.TLS != nil || (sr.URL != nil && strings.EqualFold(sr.URL.Scheme, "https")) {
		req.secure = true
		req.secureSet = true
	}
	if sr.Header != nil {
		req.headerOverlay = sr.Header.Clone()
	}
	for _, c := range sr.Cookies() {
		if c == nil {
			continue
		}
		req.SetCookie(c.Name, c.Value)
	}
	if sr.Body != nil {
		// Keep the reader lazy so middleware that must not touch the body
		// (e.g. method override via header) can still be tested.
		req.bodyReader = sr.Body
	}
	if sr.Context() != nil {
		req.SetContext(sr.Context())
	}
	return req
}

// Ctx returns the rawhttp context for this request.
// Valid only during Application.Handle → Commit. Do not store or pass to
// async work; use Method/Path/Body (owned) or copy fields into a DTO.
func (r *Request) Ctx() *rawhttp.Ctx {
	r.poisonCheck()
	if r == nil {
		return nil
	}
	return r.ctx
}

// Context returns the request-scoped context (default Background).
func (r *Request) Context() context.Context {
	r.poisonCheck()
	if r == nil || r.stdCtx == nil {
		return context.Background()
	}
	return r.stdCtx
}

// SetContext replaces the request-scoped context.
func (r *Request) SetContext(ctx context.Context) {
	r.poisonCheck()
	if r == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	r.stdCtx = ctx
}

// SetMethod overrides the HTTP method.
func (r *Request) SetMethod(method string) {
	r.poisonCheck()
	if r == nil {
		return
	}
	r.method = method
	r.methodSet = true
}

// SetPath overrides the request path (query string stripped if present).
func (r *Request) SetPath(path string) {
	r.poisonCheck()
	if r == nil {
		return
	}
	if i := strings.IndexByte(path, '?'); i >= 0 {
		r.query = path[i+1:]
		r.querySet = true
		path = path[:i]
	}
	r.path = path
	r.pathSet = true
}

// SetQueryString overrides the raw query string (without leading ?).
func (r *Request) SetQueryString(raw string) {
	r.poisonCheck()
	if r == nil {
		return
	}
	r.query = strings.TrimPrefix(raw, "?")
	r.querySet = true
}

// SetHeader sets a request header overlay value.
func (r *Request) SetHeader(key, value string) {
	r.poisonCheck()
	if r == nil {
		return
	}
	if r.headerOverlay == nil {
		r.headerOverlay = make(stdhttp.Header)
	}
	canon := textproto.CanonicalMIMEHeaderKey(key)
	delete(r.headerDeleted, canon)
	r.headerOverlay.Set(key, value)
	r.builtHeaders = nil
}

// AddHeader appends a request header overlay value.
func (r *Request) AddHeader(key, value string) {
	r.poisonCheck()
	if r == nil {
		return
	}
	if r.headerOverlay == nil {
		r.headerOverlay = make(stdhttp.Header)
	}
	canon := textproto.CanonicalMIMEHeaderKey(key)
	delete(r.headerDeleted, canon)
	r.headerOverlay.Add(key, value)
	r.builtHeaders = nil
}

// DelHeader removes a request header (overlay + hides ctx value).
func (r *Request) DelHeader(key string) {
	r.poisonCheck()
	if r == nil {
		return
	}
	canon := textproto.CanonicalMIMEHeaderKey(key)
	if r.headerOverlay != nil {
		r.headerOverlay.Del(key)
	}
	if r.headerDeleted == nil {
		r.headerDeleted = make(map[string]bool)
	}
	r.headerDeleted[canon] = true
	r.builtHeaders = nil
}

// SetCookie sets a request cookie overlay value.
func (r *Request) SetCookie(name, value string) {
	r.poisonCheck()
	if r == nil || name == "" {
		return
	}
	if r.cookieOverlay == nil {
		r.cookieOverlay = make(map[string]string)
	}
	r.cookieOverlay[name] = value
}

// SetBody replaces the request body used by Body/JSON/form parsers.
func (r *Request) SetBody(b []byte) {
	r.poisonCheck()
	if r == nil {
		return
	}
	if b == nil {
		r.bodyOverride = nil
	} else {
		r.bodyOverride = append([]byte(nil), b...)
	}
	r.bodyOverrideSet = true
	r.bodyReader = nil
	r.bodyRead = false
	r.bodyCached = nil
	r.bodyErr = nil
	r.formParsed = false
	r.multipartForm = nil
	r.jsonRead = false
	r.jsonData = nil
	r.jsonRaw = nil
}

// SetRemoteAddr overrides the direct connection address (host:port or IP).
func (r *Request) SetRemoteAddr(addr string) {
	r.poisonCheck()
	if r == nil {
		return
	}
	r.remoteAddr = addr
}

// SetHost overrides the request host.
func (r *Request) SetHost(host string) {
	r.poisonCheck()
	if r == nil {
		return
	}
	r.host = host
}

// SetSecure overrides TLS detection for this request.
func (r *Request) SetSecure(secure bool) {
	r.poisonCheck()
	if r == nil {
		return
	}
	r.secure = secure
	r.secureSet = true
}

// Method returns the HTTP method.
func (r *Request) Method() string {
	r.poisonCheck()
	if r == nil {
		return ""
	}
	if r.methodSet {
		return r.method
	}
	if r.ctx != nil && len(r.ctx.Method) > 0 {
		return string(r.ctx.Method)
	}
	return ""
}

// Path returns a copy of the request path. The string does not alias the
// request buffer, so it stays valid after the handler returns.
// PathBytes is the borrowed slice and must not be retained.
func (r *Request) Path() string {
	r.poisonCheck()
	if r == nil {
		return ""
	}
	if r.pathSet {
		return r.path
	}
	if r.ctx != nil && len(r.ctx.Path) > 0 {
		return string(r.ctx.Path)
	}
	return ""
}

// PathBytes returns the request path as a slice of the request buffer.
// It is valid only until the handler returns. Do not retain it.
// When a path overlay is set, PathBytes returns nil and Path returns the overlay.
func (r *Request) PathBytes() []byte {
	r.poisonCheck()
	if r == nil || r.pathSet || r.ctx == nil {
		return nil
	}
	return r.ctx.Path
}

// URL returns the full request URL string.
func (r *Request) URL() string {
	r.poisonCheck()
	if r == nil {
		return ""
	}
	host := r.Host()
	if host == "" {
		host = "localhost"
	}
	uri := r.RequestURI()
	if uri == "" {
		uri = "/"
	}
	return r.Scheme() + "://" + host + uri
}

// Query returns a query parameter.
func (r *Request) Query(key string, fallback ...string) string {
	r.poisonCheck()
	r.applyPendingInputTransforms()
	values := r.queryValues()
	value := ""
	if values != nil {
		value = values.Get(key)
	}
	if value == "" && len(fallback) > 0 {
		return fallback[0]
	}
	return value
}

func (r *Request) queryValues() url.Values {
	r.poisonCheck()
	qs := r.QueryString()
	if qs == "" {
		return url.Values{}
	}
	values, err := url.ParseQuery(qs)
	if err != nil {
		return url.Values{}
	}
	return values
}

// RemoteIP returns the direct connection IP (ignores forwarding headers).
func (r *Request) RemoteIP() string {
	r.poisonCheck()
	if r == nil {
		return ""
	}
	if r.ctx != nil {
		if ip := r.ctx.RemoteIP(); ip != "" {
			return ip
		}
	}
	host := r.remoteAddr
	if host == "" {
		return ""
	}
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

// IP returns the client IP. Prefers RawHTTP ClientIP (TrustedProxies), then
// the _client_ip attribute set by trustedproxy middleware, else RemoteIP.
func (r *Request) IP() string {
	r.poisonCheck()
	if r == nil {
		return ""
	}
	if r.ctx != nil {
		if ip := r.ctx.ClientIP(); ip != "" {
			return ip
		}
	}
	if v, ok := r.Get("_client_ip").(string); ok && v != "" {
		return v
	}
	return r.RemoteIP()
}

// Body returns an owned copy of the raw request body.
// It never aliases rawhttp's connection buffer (zero-copy lifecycle).
func (r *Request) Body() ([]byte, error) {
	r.poisonCheck()
	return r.readBody()
}

// SetMaxBodyBytes overrides the default JSON/raw body limit for this request.
func (r *Request) SetMaxBodyBytes(n int64) {
	r.poisonCheck()
	if r == nil {
		return
	}
	r.maxBodyBytes = n
}

func (r *Request) bodyLimit() int64 {
	r.poisonCheck()
	if r != nil && r.maxBodyBytes > 0 {
		return r.maxBodyBytes
	}
	return MaxBodyBytes()
}

func (r *Request) readBody() ([]byte, error) {
	r.poisonCheck()
	if r == nil {
		return nil, nil
	}
	if r.bodyRead {
		return r.bodyCached, r.bodyErr
	}
	r.bodyRead = true
	limit := r.bodyLimit()
	if r.bodyOverrideSet {
		if int64(len(r.bodyOverride)) > limit {
			r.bodyErr = ErrBodyTooLarge
			return nil, ErrBodyTooLarge
		}
		r.bodyCached = r.bodyOverride
		return r.bodyCached, nil
	}
	if r.ctx != nil {
		if raw := r.ctx.Body(); len(raw) > 0 {
			if int64(len(raw)) > limit {
				r.bodyErr = ErrBodyTooLarge
				return nil, ErrBodyTooLarge
			}
			// Owned copy: ctx.Body() may alias the connection read buffer
			// (rawhttp zero-copy). Do not return/cache that slice.
			r.bodyCached = append([]byte(nil), raw...)
			return r.bodyCached, nil
		}
	}
	if r.bodyReader != nil {
		raw, err := io.ReadAll(io.LimitReader(r.bodyReader, limit+1))
		_ = r.bodyReader.Close()
		r.bodyReader = nil
		if err != nil {
			r.bodyErr = err
			return nil, err
		}
		if int64(len(raw)) > limit {
			r.bodyErr = ErrBodyTooLarge
			return nil, ErrBodyTooLarge
		}
		r.bodyCached = raw
		return raw, nil
	}
	return nil, nil
}

// Route returns a route parameter.
func (r *Request) Route(key string, fallback ...string) string {
	r.poisonCheck()
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
	r.poisonCheck()
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
	r.poisonCheck()
	if r == nil {
		return
	}
	r.route = params
}

// RouteParams returns all matched route parameters.
func (r *Request) RouteParams() map[string]string {
	r.poisonCheck()
	if r == nil || len(r.route) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(r.route))
	for key, value := range r.route {
		out[key] = value
	}
	return out
}

// Set sets a request attribute.
func (r *Request) Set(key string, value any) {
	r.poisonCheck()
	if r == nil {
		return
	}
	if r.attrs == nil {
		r.attrs = make(map[string]any)
	}
	r.attrs[key] = value
}

// Get returns a request attribute.
func (r *Request) Get(key string) any {
	r.poisonCheck()
	if r == nil || r.attrs == nil {
		return nil
	}
	return r.attrs[key]
}

const sessionAttrKey = "session"

// SetSession attaches a session store as a request attribute.
func (r *Request) SetSession(store SessionStore) {
	r.poisonCheck()
	if r == nil {
		return
	}
	r.Set(sessionAttrKey, store)
}

// Session returns the session store attached to this request, if any.
func (r *Request) Session() SessionStore {
	r.poisonCheck()
	if r == nil {
		return nil
	}
	s, _ := r.Get(sessionAttrKey).(SessionStore)
	return s
}

// Cookie returns a cookie value.
func (r *Request) Cookie(name string, fallback ...string) string {
	r.poisonCheck()
	if r == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	if r.cookieOverlay != nil {
		if value, ok := r.cookieOverlay[name]; ok {
			return value
		}
	}
	if r.ctx != nil {
		if v := r.ctx.Cookie(name); len(v) > 0 {
			return string(v)
		}
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

// HasCookie reports whether a cookie is present.
func (r *Request) HasCookie(name string) bool {
	r.poisonCheck()
	if r == nil {
		return false
	}
	if r.cookieOverlay != nil {
		if _, ok := r.cookieOverlay[name]; ok {
			return true
		}
	}
	if r.ctx != nil {
		return r.ctx.Cookie(name) != nil
	}
	return false
}

// MissingCookie reports whether a cookie is absent.
func (r *Request) MissingCookie(name string) bool {
	r.poisonCheck()
	return !r.HasCookie(name)
}

// WhenHasCookie runs fn when the cookie is present.
func (r *Request) WhenHasCookie(name string, fn func(*Request)) *Request {
	r.poisonCheck()
	if r != nil && fn != nil && r.HasCookie(name) {
		fn(r)
	}
	return r
}

// WhenMissingCookie runs fn when the cookie is absent.
func (r *Request) WhenMissingCookie(name string, fn func(*Request)) *Request {
	r.poisonCheck()
	if r != nil && fn != nil && r.MissingCookie(name) {
		fn(r)
	}
	return r
}

// Cookies returns the response cookie jar for this request.
func (r *Request) Cookies() *cookie.Jar {
	r.poisonCheck()
	if r.cookies == nil {
		r.cookies = cookie.NewJar()
	}
	return r.cookies
}

// DrainCookies returns queued response cookies and clears the jar.
// A request that never queued cookies does not allocate a jar.
func (r *Request) DrainCookies() []*stdhttp.Cookie {
	r.poisonCheck()
	if r == nil || r.cookies == nil {
		return nil
	}
	out := r.cookies.Apply()
	r.cookies.Clear()
	return out
}

// IsGet reports whether the method is GET.
func (r *Request) IsGet() bool { r.poisonCheck(); return r.IsMethod("GET") }

// IsPost reports whether the method is POST.
func (r *Request) IsPost() bool { r.poisonCheck(); return r.IsMethod("POST") }

// IsPut reports whether the method is PUT.
func (r *Request) IsPut() bool { r.poisonCheck(); return r.IsMethod("PUT") }

// IsPatch reports whether the method is PATCH.
func (r *Request) IsPatch() bool { r.poisonCheck(); return r.IsMethod("PATCH") }

// IsDelete reports whether the method is DELETE.
func (r *Request) IsDelete() bool { r.poisonCheck(); return r.IsMethod("DELETE") }

// IsHead reports whether the method is HEAD.
func (r *Request) IsHead() bool { r.poisonCheck(); return r.IsMethod("HEAD") }

// IsOptions reports whether the method is OPTIONS.
func (r *Request) IsOptions() bool { r.poisonCheck(); return r.IsMethod("OPTIONS") }

// IsMethodSafe reports whether the method is GET or HEAD.
func (r *Request) IsMethodSafe() bool {
	r.poisonCheck()
	return r.IsMethod("GET", "HEAD")
}

// IsMethodIdempotent reports whether the method is GET, HEAD, PUT, DELETE, OPTIONS, or TRACE.
func (r *Request) IsMethodIdempotent() bool {
	r.poisonCheck()
	return r.IsMethod("GET", "HEAD", "PUT", "DELETE", "OPTIONS", "TRACE")
}

// HasQuery reports whether a query parameter exists (even if empty).
func (r *Request) HasQuery(key string) bool {
	r.poisonCheck()
	if r == nil {
		return false
	}
	_, ok := r.queryValues()[key]
	return ok
}

// QueryInt parses a query parameter as int with optional fallback.
func (r *Request) QueryInt(key string, fallback ...int) int {
	r.poisonCheck()
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
	r.poisonCheck()
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
	r.poisonCheck()
	switch strings.ToLower(strings.TrimSpace(r.Query(key))) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

// Port returns the request host port when present.
func (r *Request) Port() string {
	r.poisonCheck()
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
	r.poisonCheck()
	return r.Host()
}

// DecodedPath returns the URL-decoded request path.
func (r *Request) DecodedPath() string {
	r.poisonCheck()
	path := r.Path()
	decoded, err := url.PathUnescape(path)
	if err != nil {
		return path
	}
	return decoded
}

// QueryString returns the raw URL query string without leading ?.
func (r *Request) QueryString() string {
	r.poisonCheck()
	if r == nil {
		return ""
	}
	if r.querySet {
		return r.query
	}
	if r.ctx != nil {
		return string(r.ctx.Query)
	}
	return ""
}

// RequestURI returns path + query (RequestURI).
func (r *Request) RequestURI() string {
	r.poisonCheck()
	if r == nil {
		return "/"
	}
	path := r.Path()
	if path == "" {
		path = "/"
	}
	if qs := r.QueryString(); qs != "" {
		return path + "?" + qs
	}
	return path
}

// FullUrlWithQuery returns FullURL with additional/overridden query parameters.
func (r *Request) FullUrlWithQuery(extra map[string]string) string {
	r.poisonCheck()
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
	r.poisonCheck()
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
	r.poisonCheck()
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
	r.poisonCheck()
	return r.Secure()
}

func (r *Request) parseForm() error {
	r.poisonCheck()
	if r == nil {
		return nil
	}
	if r.formParsed {
		return nil
	}
	r.formParsed = true
	r.form = url.Values{}
	r.postForm = url.Values{}

	if qs := r.QueryString(); qs != "" {
		if q, err := url.ParseQuery(qs); err == nil {
			r.form = q
		}
	}

	ct := r.Header("Content-Type")
	media, _, _ := mime.ParseMediaType(ct)
	switch {
	case strings.EqualFold(media, "application/x-www-form-urlencoded"):
		body, err := r.readBody()
		if err != nil {
			return err
		}
		post, err := url.ParseQuery(string(body))
		if err != nil {
			return err
		}
		r.postForm = post
		for key, values := range post {
			r.form[key] = append(r.form[key], values...)
		}
	case strings.EqualFold(media, "multipart/form-data"):
		if err := r.parseMultipart(); err != nil {
			return err
		}
		if r.multipartForm != nil {
			for key, values := range r.multipartForm.Value {
				r.postForm[key] = append(r.postForm[key], values...)
				r.form[key] = append(r.form[key], values...)
			}
		}
	}
	return nil
}

func (r *Request) ensureForm() {
	r.poisonCheck()
	_ = r.parseForm()
	if r.form == nil {
		r.form = url.Values{}
	}
	if r.postForm == nil {
		r.postForm = url.Values{}
	}
}
