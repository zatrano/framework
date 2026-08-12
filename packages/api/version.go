package api

import (
	"strings"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

const (
	HeaderVersion = "X-API-Version"
	AttrVersion   = "api_version"
)

// Version groups routes under /api/{version}.
func Version(router *routing.Router, version string, fn func(api *routing.Router), middleware ...routing.MiddlewareFunc) {
	version = strings.Trim(version, "/")
	layers := append([]routing.MiddlewareFunc{SetVersion(version)}, middleware...)
	router.Group("/api/"+version, fn, layers...)
}

// SetVersion stores the API version on the request.
func SetVersion(version string) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
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

// FromRequest resolves the API version from attributes or headers.
func FromRequest(req *http.Request, fallback ...string) string {
	if value, ok := req.Get(AttrVersion).(string); ok && value != "" {
		return value
	}
	if header := req.Header(HeaderVersion); header != "" {
		return header
	}
	accept := req.Header("Accept")
	if strings.Contains(accept, "vnd.zatrano.") {
		// application/vnd.zatrano.v1+json
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
func RequireVersion(expected string) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
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
