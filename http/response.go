package http

import (
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"os"
	"path/filepath"
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

// ContentType returns the content type.
func (r *Response) ContentType() string {
	return r.contentType
}

// Text creates a plain text response.
func Text(body string) *Response {
	return &Response{
		status:      stdhttp.StatusOK,
		content:     []byte(body),
		contentType: "text/plain; charset=utf-8",
		headers:     make(stdhttp.Header),
	}
}

// HTML creates an HTML response.
func HTML(body string) *Response {
	return &Response{
		status:      stdhttp.StatusOK,
		content:     []byte(body),
		contentType: "text/html; charset=utf-8",
		headers:     make(stdhttp.Header),
	}
}

// JSON creates a JSON response.
func JSON(data any) *Response {
	payload, err := json.Marshal(data)
	if err != nil {
		return &Response{
			status:      stdhttp.StatusInternalServerError,
			content:     []byte(`{"message":"failed to encode json"}`),
			contentType: "application/json",
			headers:     make(stdhttp.Header),
			err:         err,
		}
	}
	return &Response{
		status:      stdhttp.StatusOK,
		content:     payload,
		contentType: "application/json",
		headers:     make(stdhttp.Header),
	}
}

// Created creates a 201 JSON response.
func Created(data any) *Response {
	return JSON(data).Status(stdhttp.StatusCreated)
}

// Accepted creates a 202 JSON response.
func Accepted(data any) *Response {
	return JSON(data).Status(stdhttp.StatusAccepted)
}

// BadRequest creates a 400 JSON message response.
func BadRequest(message ...string) *Response {
	return jsonStatusMessage(stdhttp.StatusBadRequest, "Bad Request", message...)
}

// Unauthorized creates a 401 JSON message response.
func Unauthorized(message ...string) *Response {
	return jsonStatusMessage(stdhttp.StatusUnauthorized, "Unauthorized", message...)
}

// NotFound creates a 404 JSON message response.
func NotFound(message ...string) *Response {
	return jsonStatusMessage(stdhttp.StatusNotFound, "Not Found", message...)
}

// Forbidden creates a 403 JSON message response.
func Forbidden(message ...string) *Response {
	return jsonStatusMessage(stdhttp.StatusForbidden, "Forbidden", message...)
}

// Conflict creates a 409 JSON message response.
func Conflict(message ...string) *Response {
	return jsonStatusMessage(stdhttp.StatusConflict, "Conflict", message...)
}

// Gone creates a 410 JSON message response.
func Gone(message ...string) *Response {
	return jsonStatusMessage(stdhttp.StatusGone, "Gone", message...)
}

// Unprocessable creates a 422 JSON message response.
func Unprocessable(message ...string) *Response {
	return jsonStatusMessage(stdhttp.StatusUnprocessableEntity, "Unprocessable Entity", message...)
}

// MethodNotAllowed creates a 405 JSON message response.
func MethodNotAllowed(message ...string) *Response {
	return jsonStatusMessage(stdhttp.StatusMethodNotAllowed, "Method Not Allowed", message...)
}

// PaymentRequired creates a 402 JSON message response.
func PaymentRequired(message ...string) *Response {
	return jsonStatusMessage(stdhttp.StatusPaymentRequired, "Payment Required", message...)
}

// TooManyRequests creates a 429 JSON message response.
func TooManyRequests(message ...string) *Response {
	return jsonStatusMessage(stdhttp.StatusTooManyRequests, "Too Many Requests", message...)
}

// ServiceUnavailable creates a 503 JSON message response.
func ServiceUnavailable(message ...string) *Response {
	return jsonStatusMessage(stdhttp.StatusServiceUnavailable, "Service Unavailable", message...)
}

func jsonStatusMessage(status int, fallback string, message ...string) *Response {
	msg := fallback
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	return JSON(map[string]any{"message": msg}).Status(status)
}

// NoContent creates an empty 204 response.
func NoContent() *Response {
	return &Response{
		status:  stdhttp.StatusNoContent,
		headers: make(stdhttp.Header),
	}
}

// Redirect creates a redirect response.
func Redirect(url string, status ...int) *Response {
	code := stdhttp.StatusFound
	if len(status) > 0 {
		code = status[0]
	}
	return &Response{
		status:      code,
		redirectURL: url,
		headers:     make(stdhttp.Header),
	}
}

// View creates a view response.
func View(name string, data ...map[string]any) *Response {
	payload := map[string]any{}
	if len(data) > 0 && data[0] != nil {
		payload = data[0]
	}
	return &Response{
		status:   stdhttp.StatusOK,
		viewName: name,
		viewData: payload,
		headers:  make(stdhttp.Header),
	}
}

// File creates a file response.
func File(path string) *Response {
	return &Response{
		status:   stdhttp.StatusOK,
		filePath: path,
		headers:  make(stdhttp.Header),
	}
}

// Hijack creates a response that takes over the underlying connection.
func Hijack(fn func(w stdhttp.ResponseWriter) error) *Response {
	return &Response{
		status:  101,
		hijack:  fn,
		headers: make(stdhttp.Header),
	}
}

// Abort creates an error response with a status code.
func Abort(status int, message ...string) *Response {
	msg := stdhttp.StatusText(status)
	if len(message) > 0 {
		msg = message[0]
	}
	return &Response{
		status:      status,
		content:     []byte(msg),
		contentType: "text/plain; charset=utf-8",
		headers:     make(stdhttp.Header),
	}
}

// WriteTo writes the response to a standard HTTP response writer.
func (r *Response) WriteTo(w stdhttp.ResponseWriter) error {
	if r == nil {
		w.WriteHeader(stdhttp.StatusNoContent)
		return nil
	}

	for key, values := range r.Headers() {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	for _, cookie := range r.cookies {
		stdhttp.SetCookie(w, cookie)
	}

	if r.redirectURL != "" {
		w.Header().Set("Location", r.redirectURL)
		w.WriteHeader(r.StatusCode())
		return nil
	}

	if r.hijack != nil {
		return r.hijack(w)
	}

	if r.filePath != "" {
		info, err := os.Stat(r.filePath)
		if err != nil {
			stdhttp.Error(w, "file not found", stdhttp.StatusNotFound)
			return err
		}
		if info.IsDir() {
			stdhttp.Error(w, "cannot serve directory", stdhttp.StatusBadRequest)
			return fmt.Errorf("cannot serve directory: %s", r.filePath)
		}
		if w.Header().Get("Content-Disposition") == "" {
			w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(r.filePath))
		}
		fileReq, err := stdhttp.NewRequest(stdhttp.MethodGet, "/"+filepath.Base(r.filePath), nil)
		if err != nil {
			raw, readErr := os.ReadFile(r.filePath)
			if readErr != nil {
				stdhttp.Error(w, "file not found", stdhttp.StatusNotFound)
				return readErr
			}
			w.WriteHeader(r.StatusCode())
			_, writeErr := w.Write(raw)
			return writeErr
		}
		stdhttp.ServeFile(w, fileReq, r.filePath)
		return nil
	}

	if r.contentType != "" {
		w.Header().Set("Content-Type", r.contentType)
	}

	if r.stream != nil {
		w.WriteHeader(r.StatusCode())
		flusher, ok := w.(stdhttp.Flusher)
		if !ok {
			return fmt.Errorf("streaming is not supported")
		}
		return r.stream(w, flusher)
	}

	w.WriteHeader(r.StatusCode())
	if len(r.content) > 0 {
		_, err := w.Write(r.content)
		return err
	}
	return nil
}

// SetContent sets raw response content.
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

// IsSuccessful reports whether status is 2xx.
func (r *Response) IsSuccessful() bool {
	if r == nil {
		return false
	}
	code := r.StatusCode()
	return code >= 200 && code < 300
}

// IsOk reports whether status is 200.
func (r *Response) IsOk() bool {
	if r == nil {
		return false
	}
	return r.StatusCode() == stdhttp.StatusOK
}

// IsEmpty reports whether status is 204 or 304, or body is empty without view/file/stream.
func (r *Response) IsEmpty() bool {
	if r == nil {
		return true
	}
	code := r.StatusCode()
	if code == stdhttp.StatusNoContent || code == stdhttp.StatusNotModified {
		return true
	}
	return len(r.content) == 0 && r.filePath == "" && r.viewName == "" && r.stream == nil && r.redirectURL == ""
}

// IsRedirection reports whether status is 3xx or a redirect URL is set.
func (r *Response) IsRedirection() bool {
	if r == nil {
		return false
	}
	if r.redirectURL != "" {
		return true
	}
	code := r.StatusCode()
	return code >= 300 && code < 400
}

// IsClientError reports whether status is 4xx.
func (r *Response) IsClientError() bool {
	if r == nil {
		return false
	}
	code := r.StatusCode()
	return code >= 400 && code < 500
}

// IsServerError reports whether status is 5xx.
func (r *Response) IsServerError() bool {
	if r == nil {
		return false
	}
	code := r.StatusCode()
	return code >= 500 && code < 600
}

// IsForbidden reports whether status is 403.
func (r *Response) IsForbidden() bool {
	if r == nil {
		return false
	}
	return r.StatusCode() == stdhttp.StatusForbidden
}

// IsNotFound reports whether status is 404.
func (r *Response) IsNotFound() bool {
	if r == nil {
		return false
	}
	return r.StatusCode() == stdhttp.StatusNotFound
}

// IsUnauthorized reports whether status is 401.
func (r *Response) IsUnauthorized() bool {
	if r == nil {
		return false
	}
	return r.StatusCode() == stdhttp.StatusUnauthorized
}

// IsInformational reports whether status is 1xx.
func (r *Response) IsInformational() bool {
	if r == nil {
		return false
	}
	code := r.StatusCode()
	return code >= 100 && code < 200
}

// Failed reports whether the response is a client or server error.
func (r *Response) Failed() bool {
	return r.IsClientError() || r.IsServerError()
}

// IsJSON reports whether the content type is JSON.
func (r *Response) IsJSON() bool {
	if r == nil {
		return false
	}
	return strings.Contains(strings.ToLower(r.ContentType()), "json")
}

// IsHTML reports whether the content type is HTML.
func (r *Response) IsHTML() bool {
	if r == nil {
		return false
	}
	return strings.Contains(strings.ToLower(r.ContentType()), "html")
}

// IsText reports whether the content type is plain text.
func (r *Response) IsText() bool {
	if r == nil {
		return false
	}
	ct := strings.ToLower(r.ContentType())
	return strings.HasPrefix(ct, "text/plain")
}

// ContentLength returns the body byte length.
func (r *Response) ContentLength() int {
	if r == nil {
		return 0
	}
	return len(r.content)
}

// OK creates a 200 JSON response.
func OK(data any) *Response {
	return JSON(data)
}

// Empty is an alias for NoContent.
func Empty() *Response {
	return NoContent()
}

// Bytes creates a response from raw bytes.
func Bytes(content []byte, contentType string) *Response {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &Response{
		status:      stdhttp.StatusOK,
		content:     content,
		contentType: contentType,
		headers:     make(stdhttp.Header),
	}
}

// XML creates an XML text response.
func XML(body string) *Response {
	return &Response{
		status:      stdhttp.StatusOK,
		content:     []byte(body),
		contentType: "application/xml; charset=utf-8",
		headers:     make(stdhttp.Header),
	}
}

// JSONP wraps JSON data in a callback for JSONP responses.
func JSONP(callback string, data any) *Response {
	payload, err := json.Marshal(data)
	if err != nil {
		payload = []byte("null")
	}
	if callback == "" {
		callback = "callback"
	}
	callback = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_' || r == '$' || r == '.':
			return r
		default:
			return -1
		}
	}, callback)
	if callback == "" {
		callback = "callback"
	}
	body := callback + "(" + string(payload) + ");"
	return &Response{
		status:      stdhttp.StatusOK,
		content:     []byte(body),
		contentType: "application/javascript; charset=utf-8",
		headers:     make(stdhttp.Header),
	}
}

// Make creates a response with explicit status, body, and content type.
func Make(status int, content []byte, contentType string) *Response {
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	return &Response{
		status:      status,
		content:     content,
		contentType: contentType,
		headers:     make(stdhttp.Header),
	}
}

// InternalServerError creates a 500 JSON message response.
func InternalServerError(message ...string) *Response {
	return jsonStatusMessage(stdhttp.StatusInternalServerError, "Internal Server Error", message...)
}

// NotModified creates a 304 response.
func NotModified() *Response {
	return &Response{
		status:  stdhttp.StatusNotModified,
		headers: make(stdhttp.Header),
	}
}

// PartialContent creates a 206 response with raw content.
func PartialContent(content []byte, contentType string) *Response {
	return Bytes(content, contentType).Status(stdhttp.StatusPartialContent)
}

// Found creates a 302 redirect.
func Found(url string) *Response {
	return Redirect(url, stdhttp.StatusFound)
}

// RedirectRoute redirects to a path produced for a named route (caller supplies path).
// Prefer routing.Router.RedirectRoute when a router is available.
func RedirectRoute(path string, status ...int) *Response {
	if path == "" {
		path = "/"
	}
	return Redirect(path, status...)
}

// StreamDownload streams content with an attachment disposition.
func StreamDownload(filename string, writer StreamWriter) *Response {
	resp := Stream("application/octet-stream", writer)
	return resp.AsDownload(filename)
}

// SeeOther creates a 303 redirect.
func SeeOther(url string) *Response {
	return Redirect(url, stdhttp.StatusSeeOther)
}

// TemporaryRedirect creates a 307 redirect.
func TemporaryRedirect(url string) *Response {
	return Redirect(url, stdhttp.StatusTemporaryRedirect)
}
