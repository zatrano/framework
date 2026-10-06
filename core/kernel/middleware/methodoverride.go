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

// MethodOverrideFromHeader is the header-time half of ApplyMethodOverride.
// resolved is the method Dispatch will use when the body cannot change it.
// ambiguous is true only for POST application/x-www-form-urlencoded with no
// override header: _method in the body may still select PUT, PATCH, or DELETE.
// JSON, multipart, and query strings are not override sources. A present
// override header never reads the body.
func MethodOverrideFromHeader(method, overrideHeader, contentType string) (resolved string, ambiguous bool) {
	resolved = strings.ToUpper(strings.TrimSpace(method))
	if resolved != "POST" {
		return resolved, false
	}
	if header := strings.TrimSpace(overrideHeader); header != "" {
		if override := allowedMethodOverride(header); override != "" {
			return override, false
		}
		return resolved, false
	}
	media, _, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.EqualFold(media, "application/x-www-form-urlencoded") {
		return resolved, false
	}
	return resolved, true
}

// ApplyMethodOverride rewrites POST using X-HTTP-Method-Override, then
// application/x-www-form-urlencoded _method. JSON, multipart, and query
// are not override sources. A present override header never reads the body.
func ApplyMethodOverride(req *http.Request) {
	if req == nil {
		return
	}
	method := req.Method()
	if !strings.EqualFold(method, "POST") {
		return
	}
	override, _ := req.HeaderValue("X-HTTP-Method-Override")
	contentType, _ := req.HeaderValue("Content-Type")
	resolved, ambiguous := MethodOverrideFromHeader(method, override, contentType)
	if !strings.EqualFold(resolved, req.Method()) {
		req.SetMethod(resolved)
	}
	if !ambiguous {
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
