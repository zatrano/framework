package middleware

import (
	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

// SecurityHeaders adds common browser security headers.
func SecurityHeaders(next routing.HandlerFunc) routing.HandlerFunc {
	return SecurityHeadersWith(SecurityHeaderConfig{})(next)
}

// SecurityHeaderConfig customizes security headers.
type SecurityHeaderConfig struct {
	FrameOptions       string
	ContentTypeOptions string
	ReferrerPolicy     string
	PermissionsPolicy  string
	HSTS               string // explicit value; sent only when the request is HTTPS
	// EnableHSTSOnHTTPS emits max-age-only HSTS when the request is HTTPS
	// (TLS or trusted forwarded proto). includeSubDomains/preload are not defaulted.
	EnableHSTSOnHTTPS bool
}

const defaultHSTS = "max-age=31536000"

// SecurityHeadersWith adds configured security headers to every response.
func SecurityHeadersWith(cfg SecurityHeaderConfig) routing.MiddlewareFunc {
	if cfg.FrameOptions == "" {
		cfg.FrameOptions = "SAMEORIGIN"
	}
	if cfg.ContentTypeOptions == "" {
		cfg.ContentTypeOptions = "nosniff"
	}
	if cfg.ReferrerPolicy == "" {
		cfg.ReferrerPolicy = "strict-origin-when-cross-origin"
	}
	if cfg.PermissionsPolicy == "" {
		cfg.PermissionsPolicy = "geolocation=(), microphone=(), camera=()"
	}

	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			resp := next(req)
			if resp == nil {
				return resp
			}
			resp.Header("X-Frame-Options", cfg.FrameOptions)
			resp.Header("X-Content-Type-Options", cfg.ContentTypeOptions)
			resp.Header("Referrer-Policy", cfg.ReferrerPolicy)
			resp.Header("Permissions-Policy", cfg.PermissionsPolicy)
			if req != nil && req.Secure() {
				hsts := cfg.HSTS
				if hsts == "" && cfg.EnableHSTSOnHTTPS {
					hsts = defaultHSTS
				}
				if hsts != "" {
					resp.Header("Strict-Transport-Security", hsts)
				}
			}
			return resp
		}
	}
}
