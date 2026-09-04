package middleware

import (
	"strings"

	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

// ApplyMethodOverride rewrites POST requests using _method or X-HTTP-Method-Override.
// Call this before route matching so spoofed methods can hit PUT/PATCH/DELETE routes.
func ApplyMethodOverride(req *http.Request) {
	if req == nil || req.Raw() == nil {
		return
	}
	if !strings.EqualFold(req.Raw().Method, "POST") {
		return
	}
	override := strings.TrimSpace(req.Input("_method"))
	if override == "" {
		override = strings.TrimSpace(req.Header("X-HTTP-Method-Override"))
	}
	override = strings.ToUpper(override)
	switch override {
	case "PUT", "PATCH", "DELETE":
		req.Raw().Method = override
	}
}

// MethodOverride rewrites POST requests using _method or X-HTTP-Method-Override.
func MethodOverride(next routing.HandlerFunc) routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		ApplyMethodOverride(req)
		return next(req)
	}
}
