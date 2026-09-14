package http

import (
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"os"
	"path/filepath"
	"strings"
)

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
		if !r.publicFile && w.Header().Get("Content-Disposition") == "" {
			w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(r.filePath))
		}
		fileReq := r.fileHTTPReq
		if fileReq == nil {
			var err error
			fileReq, err = stdhttp.NewRequest(stdhttp.MethodGet, "/"+filepath.Base(r.filePath), nil)
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
