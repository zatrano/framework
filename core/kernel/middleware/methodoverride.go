package middleware

import (
	"mime"
	"strings"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
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
	if req == nil {
		return
	}
	if !strings.EqualFold(req.Method(), "POST") {
		return
	}
	if header := strings.TrimSpace(req.Header("X-HTTP-Method-Override")); header != "" {
		if override := allowedMethodOverride(header); override != "" {
			req.SetMethod(override)
		}
		return
	}
	media, _, err := mime.ParseMediaType(req.Header("Content-Type"))
	if err != nil || !strings.EqualFold(media, "application/x-www-form-urlencoded") {
		return
	}
	if override := allowedMethodOverride(req.PostForm("_method")); override != "" {
		req.SetMethod(override)
	}
}

// MethodOverride rewrites POST requests using the same rules as ApplyMethodOverride.
func MethodOverride(next routing.HandlerFunc) routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		ApplyMethodOverride(req)
		return next(req)
	}
}
