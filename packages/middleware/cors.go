package middleware

import (
	"strconv"
	"strings"

	"github.com/zatrano/framework/packages/env"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// CORSConfig configures Cross-Origin Resource Sharing headers.
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     string
	AllowHeaders     string
	ExposeHeaders    string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a permissive local-development CORS policy.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		AllowHeaders: "Content-Type, Authorization, X-Requested-With, X-CSRF-TOKEN, X-Idempotency-Key",
		MaxAge:       600,
	}
}

// CORSFromEnv builds CORS middleware from environment variables.
func CORSFromEnv() routing.MiddlewareFunc {
	cfg := DefaultCORSConfig()
	if raw := env.Get("CORS_ALLOWED_ORIGINS"); raw != "" {
		parts := strings.Split(raw, ",")
		origins := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				origins = append(origins, p)
			}
		}
		if len(origins) > 0 {
			cfg.AllowOrigins = origins
		}
	}
	if v := env.Get("CORS_ALLOWED_METHODS"); v != "" {
		cfg.AllowMethods = v
	}
	if v := env.Get("CORS_ALLOWED_HEADERS"); v != "" {
		cfg.AllowHeaders = v
	}
	if v := env.Get("CORS_EXPOSE_HEADERS"); v != "" {
		cfg.ExposeHeaders = v
	}
	cfg.AllowCredentials = env.GetBool("CORS_ALLOW_CREDENTIALS", false)
	if v := env.Get("CORS_MAX_AGE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxAge = n
		}
	}
	return CORSWith(cfg)
}

// CORSWith returns CORS middleware for the given config.
func CORSWith(cfg CORSConfig) routing.MiddlewareFunc {
	if len(cfg.AllowOrigins) == 0 {
		cfg.AllowOrigins = []string{"*"}
	}
	if cfg.AllowMethods == "" {
		cfg.AllowMethods = DefaultCORSConfig().AllowMethods
	}
	if cfg.AllowHeaders == "" {
		cfg.AllowHeaders = DefaultCORSConfig().AllowHeaders
	}

	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			origin := req.Header("Origin")
			allowOrigin := resolveOrigin(cfg.AllowOrigins, origin)

			apply := func(resp *http.Response) *http.Response {
				if resp == nil {
					return resp
				}
				if allowOrigin != "" {
					resp.Header("Access-Control-Allow-Origin", allowOrigin)
				}
				resp.Header("Access-Control-Allow-Methods", cfg.AllowMethods)
				resp.Header("Access-Control-Allow-Headers", cfg.AllowHeaders)
				if cfg.ExposeHeaders != "" {
					resp.Header("Access-Control-Expose-Headers", cfg.ExposeHeaders)
				}
				if cfg.AllowCredentials && allowOrigin != "*" {
					resp.Header("Access-Control-Allow-Credentials", "true")
				}
				if cfg.MaxAge > 0 {
					resp.Header("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}
				if allowOrigin != "*" && allowOrigin != "" {
					resp.Header("Vary", "Origin")
				}
				return resp
			}

			if req.Method() == "OPTIONS" {
				return apply(http.NoContent())
			}
			return apply(next(req))
		}
	}
}

func resolveOrigin(allowed []string, requestOrigin string) string {
	for _, o := range allowed {
		if o == "*" {
			return "*"
		}
		if requestOrigin != "" && strings.EqualFold(o, requestOrigin) {
			return requestOrigin
		}
	}
	if len(allowed) == 1 && allowed[0] != "*" {
		return allowed[0]
	}
	return ""
}
