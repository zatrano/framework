package exceptions

import (
	"fmt"
	"html"
	"log"
	"runtime/debug"

	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

// HTTPError is an error with an HTTP status.
type HTTPError struct {
	Status  int
	Message string
	Cause   error
}

func (e *HTTPError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return fmt.Sprintf("HTTP %d", e.Status)
}

func (e *HTTPError) Unwrap() error { return e.Cause }

// NotFound creates a 404 error.
func NotFound(message ...string) *HTTPError {
	msg := "Not Found"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	return &HTTPError{Status: 404, Message: msg}
}

// Forbidden creates a 403 error.
func Forbidden(message ...string) *HTTPError {
	msg := "Forbidden"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	return &HTTPError{Status: 403, Message: msg}
}

// Unauthorized creates a 401 error.
func Unauthorized(message ...string) *HTTPError {
	msg := "Unauthenticated."
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	return &HTTPError{Status: 401, Message: msg}
}

// Abort creates an HTTP error with status.
func Abort(status int, message string) *HTTPError {
	return &HTTPError{Status: status, Message: message}
}

// Reporter receives reported exceptions.
type Reporter func(err error, req *http.Request)

// Renderer customizes HTTP rendering for a status code.
type Renderer func(req *http.Request, err error) *http.Response

// Handler reports and renders exceptions.
type Handler struct {
	debug     bool
	reporters []Reporter
	renderers map[int]Renderer
}

// New creates an exception handler.
func New(debug bool) *Handler {
	return &Handler{
		debug:     debug,
		reporters: make([]Reporter, 0),
		renderers: make(map[int]Renderer),
	}
}

// ReportUsing registers a reporter.
func (h *Handler) ReportUsing(fn Reporter) {
	h.reporters = append(h.reporters, fn)
}

// RenderUsing registers a status-specific renderer.
func (h *Handler) RenderUsing(status int, fn Renderer) {
	h.renderers[status] = fn
}

// Report notifies all reporters.
func (h *Handler) Report(err error, req *http.Request) {
	if err == nil {
		return
	}
	for _, reporter := range h.reporters {
		reporter(err, req)
	}
	if len(h.reporters) == 0 {
		log.Printf("exception: %v", err)
	}
}

// Render converts an error into an HTTP response.
func (h *Handler) Render(req *http.Request, err error) *http.Response {
	if err == nil {
		return nil
	}
	status := 500
	message := "Server Error"
	if httpErr, ok := err.(*HTTPError); ok {
		status = httpErr.Status
		message = httpErr.Error()
	} else {
		message = err.Error()
	}
	if renderer, ok := h.renderers[status]; ok {
		return renderer(req, err)
	}

	wantsJSON := req != nil && req.WantsJSON()
	if wantsJSON {
		payload := map[string]any{"message": message}
		if h.debug && status >= 500 {
			payload["exception"] = fmt.Sprintf("%v", err)
		}
		return http.JSON(payload).Status(status)
	}

	title := httpTitle(status)
	body := message
	if h.debug && status >= 500 {
		body = fmt.Sprintf("%s\n\n%v\n\n%s", message, err, string(debug.Stack()))
	}
	page := fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>%d %s</title>
<style>body{font-family:ui-sans-serif,system-ui;background:#0b1220;color:#e8eef8;padding:2rem;max-width:880px;margin:0 auto}
h1{color:#3dd6c6}pre{white-space:pre-wrap;background:#121a2b;padding:1rem;border-radius:8px;overflow:auto}</style></head>
<body><h1>%d %s</h1><pre>%s</pre></body></html>`, status, title, status, title, html.EscapeString(body))
	return http.HTML(page).Status(status)
}

// Middleware recovers panics and renders them through the handler.
func (h *Handler) Middleware() routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) (resp *http.Response) {
			defer func() {
				if recovered := recover(); recovered != nil {
					var err error
					switch v := recovered.(type) {
					case error:
						err = v
					default:
						err = fmt.Errorf("%v", v)
					}
					h.Report(err, req)
					resp = h.Render(req, err)
				}
			}()
			return next(req)
		}
	}
}

func httpTitle(status int) string {
	switch status {
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 419:
		return "Page Expired"
	case 429:
		return "Too Many Requests"
	case 503:
		return "Service Unavailable"
	default:
		if status >= 500 {
			return "Server Error"
		}
		return "Error"
	}
}
