package middleware

import (
	"mime"
	"strings"

	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

func allowedMethodOverride(method string) string {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "PUT", "PATCH", "DELETE":
		return strings.ToUpper(strings.TrimSpace(method))
	default:
		return ""
	}
}

// ApplyMethodOverride rewrites POST using X-HTTP-Method-Override, then
// application/x-www-form-urlencoded _method. JSON, multipart, and query
// are not override sources. A present override header never reads the body.
func ApplyMethodOverride(req *http.Request) {
	if req == nil || req.Raw() == nil {
		return
	}
	raw := req.Raw()
	if !strings.EqualFold(raw.Method, "POST") {
		return
	}
	if header := strings.TrimSpace(raw.Header.Get("X-HTTP-Method-Override")); header != "" {
		if override := allowedMethodOverride(header); override != "" {
			raw.Method = override
		}
		return
	}
	media, _, err := mime.ParseMediaType(raw.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(media, "application/x-www-form-urlencoded") {
		return
	}
	if err := raw.ParseForm(); err != nil {
		return
	}
	if override := allowedMethodOverride(raw.PostForm.Get("_method")); override != "" {
		raw.Method = override
	}
}

// MethodOverride rewrites POST requests using the same rules as ApplyMethodOverride.
func MethodOverride(next routing.HandlerFunc) routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		ApplyMethodOverride(req)
		return next(req)
	}
}
