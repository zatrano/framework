package routing

import (
	"strings"

	"github.com/zatrano/framework/v2/kernel/http"
)

const (
	// HeaderVersion is set on versioned API responses.
	HeaderVersion = "X-API-Version"
	// AttrVersion is the request attribute holding the active API version.
	AttrVersion = "api_version"
)

// Version groups routes under /api/{version} and stamps X-API-Version.
func Version(router *Router, version string, fn func(api *Router), middleware ...MiddlewareFunc) {
	if router == nil || fn == nil {
		return
	}
	version = strings.Trim(version, "/")
	layers := append([]MiddlewareFunc{SetVersion(version)}, middleware...)
	router.Group("/api/"+version, fn, layers...)
}

// SetVersion stores the API version on the request and response.
func SetVersion(version string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(req *http.Request) *http.Response {
			req.Set(AttrVersion, version)
			resp := next(req)
			if resp != nil {
				resp.Header(HeaderVersion, version)
			}
			return resp
		}
	}
}

// FromRequest resolves the API version from attributes, X-API-Version, or Accept.
func FromRequest(req *http.Request, fallback ...string) string {
	if req == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	if value, ok := req.Get(AttrVersion).(string); ok && value != "" {
		return value
	}
	if header := req.Header(HeaderVersion); header != "" {
		return header
	}
	accept := req.Header("Accept")
	if strings.Contains(accept, "vnd.zatrano.") {
		parts := strings.Split(accept, "vnd.zatrano.")
		if len(parts) > 1 {
			rest := parts[1]
			rest = strings.Split(rest, "+")[0]
			rest = strings.Split(rest, ";")[0]
			return strings.TrimSpace(rest)
		}
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

// RequireVersion rejects requests that do not match the expected version.
func RequireVersion(expected string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(req *http.Request) *http.Response {
			actual := FromRequest(req)
			if actual == "" {
				actual = expected
				req.Set(AttrVersion, expected)
			}
			if actual != expected {
				return http.JSON(map[string]any{
					"message":  "Unsupported API version",
					"expected": expected,
					"actual":   actual,
				}).Status(406)
			}
			return next(req)
		}
	}
}
