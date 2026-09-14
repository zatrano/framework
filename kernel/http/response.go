package http

import (
	"fmt"
	stdhttp "net/http"
	"strings"
	"time"
)

// Response represents an HTTP response to be sent.
type Response struct {
	status      int
	headers     stdhttp.Header
	cookies     []*stdhttp.Cookie
	content     []byte
	contentType string
	filePath    string
	fileHTTPReq *stdhttp.Request
	publicFile  bool
	redirectURL string
	viewName    string
	viewData    map[string]any
	err         error
	stream      StreamWriter
	hijack      func(w stdhttp.ResponseWriter) error
}

// Status sets the response status code.
func (r *Response) Status(code int) *Response {
	r.status = code
	return r
}

// Header sets a response header.
func (r *Response) Header(key, value string) *Response {
	if r.headers == nil {
		r.headers = make(stdhttp.Header)
	}
	r.headers.Set(key, value)
	return r
}

// WithCookie adds a cookie to the response.
func (r *Response) WithCookie(cookie *stdhttp.Cookie) *Response {
	if r == nil {
		return nil
	}
	r.cookies = append(r.cookies, cookie)
	return r
}

// WithHeaders sets multiple response headers.
func (r *Response) WithHeaders(headers map[string]string) *Response {
	for key, value := range headers {
		r.Header(key, value)
	}
	return r
}

// NoCache marks the response as non-cacheable.
func (r *Response) NoCache() *Response {
	if r == nil {
		return nil
	}
	return r.Header("Cache-Control", "no-cache, no-store, must-revalidate").
		Header("Pragma", "no-cache").
		Header("Expires", "0")
}

// CacheFor sets a public Cache-Control max-age from duration.
func (r *Response) CacheFor(d time.Duration) *Response {
	if r == nil {
		return nil
	}
	if d < 0 {
		d = 0
	}
	return r.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", int(d.Seconds())))
}

// PrivateCache sets a private Cache-Control max-age from duration.
func (r *Response) PrivateCache(d time.Duration) *Response {
	if r == nil {
		return nil
	}
	if d < 0 {
		d = 0
	}
	return r.Header("Cache-Control", fmt.Sprintf("private, max-age=%d", int(d.Seconds())))
}

// Vary sets the Vary response header.
func (r *Response) Vary(headers ...string) *Response {
	if r == nil || len(headers) == 0 {
		return r
	}
	return r.Header("Vary", strings.Join(headers, ", "))
}

// WithoutHeader removes a response header.
func (r *Response) WithoutHeader(key string) *Response {
	if r.headers != nil {
		r.headers.Del(key)
	}
	return r
}

// Cookie sets a simple cookie (optional MaxAge in seconds).
func (r *Response) Cookie(name, value string, maxAge ...int) *Response {
	cookie := &stdhttp.Cookie{
		Name:  name,
		Value: value,
		Path:  "/",
	}
	if len(maxAge) > 0 {
		cookie.MaxAge = maxAge[0]
	}
	return r.WithCookie(cookie)
}

// WithoutCookie expires a cookie by name.
func (r *Response) WithoutCookie(name string) *Response {
	return r.WithCookie(&stdhttp.Cookie{
		Name:   name,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// Content returns the response body bytes.
func (r *Response) Content() []byte {
	return r.content
}

// StatusCode returns the status code.
func (r *Response) StatusCode() int {
	if r.status == 0 {
		return stdhttp.StatusOK
	}
	return r.status
}

// ViewName returns the view name when rendering a template.
func (r *Response) ViewName() string {
	return r.viewName
}

// ViewData returns the view data.
func (r *Response) ViewData() map[string]any {
	return r.viewData
}

// Error returns a response-level error, if any.
func (r *Response) Error() error {
	return r.err
}

// IsRedirect reports whether the response is a redirect.
func (r *Response) IsRedirect() bool {
	return r.redirectURL != ""
}

// RedirectURL returns the redirect target.
func (r *Response) RedirectURL() string {
	return r.redirectURL
}

// FilePath returns a file download path, if any.
func (r *Response) FilePath() string {
	return r.filePath
}

// Headers returns response headers.
func (r *Response) Headers() stdhttp.Header {
	if r.headers == nil {
		r.headers = make(stdhttp.Header)
	}
	return r.headers
}

// Cookies returns response cookies.
func (r *Response) Cookies() []*stdhttp.Cookie {
	return r.cookies
}

// ReplaceCookies replaces the queued response cookies.
func (r *Response) ReplaceCookies(cookies []*stdhttp.Cookie) *Response {
	if r == nil {
		return nil
	}
	r.cookies = cookies
	return r
}

// ContentType returns the content type.
func (r *Response) ContentType() string {
	return r.contentType
}

func (r *Response) SetContent(content []byte, contentType string) *Response {
	r.content = content
	r.contentType = contentType
	return r
}

// WithContentType sets the response content type.
func (r *Response) WithContentType(contentType string) *Response {
	if r == nil {
		return nil
	}
	r.contentType = contentType
	return r
}

// Charset appends or replaces charset on the content type (defaults to text/plain).
func (r *Response) Charset(charset string) *Response {
	if r == nil {
		return nil
	}
	if charset == "" {
		charset = "utf-8"
	}
	base := r.contentType
	if base == "" {
		base = "text/plain"
	}
	if i := strings.Index(base, ";"); i >= 0 {
		base = strings.TrimSpace(base[:i])
	}
	r.contentType = base + "; charset=" + charset
	return r
}

// AppendHeader adds a header value without replacing existing ones.
func (r *Response) AppendHeader(key, value string) *Response {
	if r == nil {
		return nil
	}
	if r.headers == nil {
		r.headers = make(stdhttp.Header)
	}
	r.headers.Add(key, value)
	return r
}

// WithoutHeaders removes multiple response headers.
func (r *Response) WithoutHeaders(keys ...string) *Response {
	if r == nil {
		return nil
	}
	for _, key := range keys {
		r.WithoutHeader(key)
	}
	return r
}

// HasHeader reports whether a response header is set.
func (r *Response) HasHeader(key string) bool {
	if r == nil || r.headers == nil {
		return false
	}
	return r.headers.Get(key) != ""
}

// GetHeader returns a response header value.
func (r *Response) GetHeader(key string, fallback ...string) string {
	if r == nil || r.headers == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	value := r.headers.Get(key)
	if value == "" && len(fallback) > 0 {
		return fallback[0]
	}
	return value
}

// Location sets the Location header.
func (r *Response) Location(url string) *Response {
	return r.Header("Location", url)
}

// Allow sets the Allow header (HTTP methods).
func (r *Response) Allow(methods ...string) *Response {
	if r == nil || len(methods) == 0 {
		return r
	}
	return r.Header("Allow", strings.Join(methods, ", "))
}

// ContentLanguage sets the Content-Language header.
func (r *Response) ContentLanguage(lang string) *Response {
	return r.Header("Content-Language", lang)
}

// ContentEncoding sets the Content-Encoding header.
func (r *Response) ContentEncoding(encoding string) *Response {
	return r.Header("Content-Encoding", encoding)
}

// WwwAuthenticate sets the WWW-Authenticate header.
func (r *Response) WwwAuthenticate(value string) *Response {
	return r.Header("WWW-Authenticate", value)
}

// StaleWhileRevalidate appends stale-while-revalidate to Cache-Control.
func (r *Response) StaleWhileRevalidate(seconds int) *Response {
	if seconds < 0 {
		seconds = 0
	}
	return r.appendCacheDirective(fmt.Sprintf("stale-while-revalidate=%d", seconds))
}

// StaleIfError appends stale-if-error to Cache-Control.
func (r *Response) StaleIfError(seconds int) *Response {
	if seconds < 0 {
		seconds = 0
	}
	return r.appendCacheDirective(fmt.Sprintf("stale-if-error=%d", seconds))
}

// ETag sets the ETag header.
func (r *Response) ETag(tag string) *Response {
	return r.Header("ETag", tag)
}

// LastModified sets the Last-Modified header.
func (r *Response) LastModified(t time.Time) *Response {
	return r.Header("Last-Modified", t.UTC().Format(stdhttp.TimeFormat))
}

// ExpiresAt sets the Expires header.
func (r *Response) ExpiresAt(t time.Time) *Response {
	return r.Header("Expires", t.UTC().Format(stdhttp.TimeFormat))
}

// MaxAge sets Cache-Control max-age in seconds (public).
func (r *Response) MaxAge(seconds int) *Response {
	if r == nil {
		return nil
	}
	if seconds < 0 {
		seconds = 0
	}
	return r.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", seconds))
}

// SharedMaxAge sets Cache-Control s-maxage.
func (r *Response) SharedMaxAge(seconds int) *Response {
	if r == nil {
		return nil
	}
	if seconds < 0 {
		seconds = 0
	}
	current := r.GetHeader("Cache-Control")
	directive := fmt.Sprintf("s-maxage=%d", seconds)
	if current == "" {
		return r.Header("Cache-Control", directive)
	}
	return r.Header("Cache-Control", current+", "+directive)
}

// MustRevalidate appends must-revalidate to Cache-Control.
func (r *Response) MustRevalidate() *Response {
	return r.appendCacheDirective("must-revalidate")
}

// Immutable appends immutable to Cache-Control.
func (r *Response) Immutable() *Response {
	return r.appendCacheDirective("immutable")
}

// Public appends public to Cache-Control.
func (r *Response) Public() *Response {
	return r.appendCacheDirective("public")
}

// Private appends private to Cache-Control.
func (r *Response) Private() *Response {
	return r.appendCacheDirective("private")
}

func (r *Response) appendCacheDirective(directive string) *Response {
	if r == nil {
		return nil
	}
	current := r.GetHeader("Cache-Control")
	if current == "" {
		return r.Header("Cache-Control", directive)
	}
	if strings.Contains(current, directive) {
		return r
	}
	return r.Header("Cache-Control", current+", "+directive)
}

// CookieOptions configures a response cookie.
type CookieOptions struct {
	MaxAge   int
	Path     string
	Domain   string
	Secure   bool
	HTTPOnly bool
	SameSite stdhttp.SameSite
}

// WithCookieOptions sets a cookie with full options.
func (r *Response) WithCookieOptions(name, value string, opts CookieOptions) *Response {
	if r == nil {
		return nil
	}
	path := opts.Path
	if path == "" {
		path = "/"
	}
	cookie := &stdhttp.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   opts.Domain,
		MaxAge:   opts.MaxAge,
		Secure:   opts.Secure,
		HttpOnly: opts.HTTPOnly,
		SameSite: opts.SameSite,
	}
	return r.WithCookie(cookie)
}

// CookieForever sets a long-lived cookie (~5 years).
func (r *Response) CookieForever(name, value string) *Response {
	return r.Cookie(name, value, 5*365*24*60*60)
}

// CookieMinutes sets a cookie with max-age in minutes.
func (r *Response) CookieMinutes(name, value string, minutes int) *Response {
	return r.Cookie(name, value, minutes*60)
}

// SecureCookie sets a Secure + HttpOnly cookie.
func (r *Response) SecureCookie(name, value string, maxAge ...int) *Response {
	opts := CookieOptions{Path: "/", Secure: true, HTTPOnly: true, SameSite: stdhttp.SameSiteLaxMode}
	if len(maxAge) > 0 {
		opts.MaxAge = maxAge[0]
	}
	return r.WithCookieOptions(name, value, opts)
}
