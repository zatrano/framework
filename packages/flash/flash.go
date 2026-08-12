package flash

import (
	"github.com/zatrano/framework/packages/http"
)

const (
	KeySuccess = "flash_success"
	KeyError   = "flash_error"
	KeyWarning = "flash_warning"
	KeyInfo    = "flash_info"
	KeyStatus  = "flash_status"
	KeyInput   = "_old_input"
)

// Success stores a success flash message.
func Success(req *http.Request, message string) {
	put(req, KeySuccess, message)
}

// Error stores an error flash message.
func Error(req *http.Request, message string) {
	put(req, KeyError, message)
}

// Warning stores a warning flash message.
func Warning(req *http.Request, message string) {
	put(req, KeyWarning, message)
}

// Info stores an info flash message.
func Info(req *http.Request, message string) {
	put(req, KeyInfo, message)
}

// Status stores a generic status flash message.
func Status(req *http.Request, message string) {
	put(req, KeyStatus, message)
}

// Get returns a flash value for the current request.
func Get(req *http.Request, key string) string {
	sess := req.Session()
	if sess == nil {
		return ""
	}
	if value, ok := sess.Get(key).(string); ok {
		return value
	}
	return ""
}

// Has reports whether a flash key exists.
func Has(req *http.Request, key string) bool {
	return Get(req, key) != ""
}

// All returns common flash messages.
func All(req *http.Request) map[string]string {
	return map[string]string{
		"success": Get(req, KeySuccess),
		"error":   Get(req, KeyError),
		"warning": Get(req, KeyWarning),
		"info":    Get(req, KeyInfo),
		"status":  Get(req, KeyStatus),
	}
}

// Old stores input for the next request.
func Old(req *http.Request, input map[string]string) {
	sess := req.Session()
	if sess == nil {
		return
	}
	sess.Flash(KeyInput, input)
}

// OldInput returns previously flashed input.
func OldInput(req *http.Request) map[string]string {
	sess := req.Session()
	if sess == nil {
		return map[string]string{}
	}
	raw := sess.Get(KeyInput)
	if raw == nil {
		return map[string]string{}
	}
	switch v := raw.(type) {
	case map[string]string:
		return v
	case map[string]any:
		out := make(map[string]string, len(v))
		for key, value := range v {
			out[key] = stringify(value)
		}
		return out
	default:
		return map[string]string{}
	}
}

// OldValue returns one old input value.
func OldValue(req *http.Request, key string, fallback ...string) string {
	input := OldInput(req)
	if value, ok := input[key]; ok {
		return value
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

// WithFlash redirects with a flash message.
func WithFlash(req *http.Request, key, message, redirectTo string) *http.Response {
	put(req, key, message)
	return http.Redirect(redirectTo)
}

// WithSuccess redirects with a success flash.
func WithSuccess(req *http.Request, message, redirectTo string) *http.Response {
	return WithFlash(req, KeySuccess, message, redirectTo)
}

// WithError redirects with an error flash.
func WithError(req *http.Request, message, redirectTo string) *http.Response {
	return WithFlash(req, KeyError, message, redirectTo)
}

func put(req *http.Request, key, message string) {
	sess := req.Session()
	if sess == nil {
		return
	}
	sess.Flash(key, message)
}

func stringify(value any) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}
