package middleware

import (
	"strconv"
	"strings"

	"github.com/zatrano/framework/kernel/env"
	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

// CORSConfig configures Cross-Origin Resource Sharing headers.
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     string
	AllowHeaders     string
	ExposeHeaders    string
	AllowCredentials bool
	MaxAge           int
	// Production strips wildcard origins. Set from the application
	// environment snapshot, not a live APP_ENV parse.
	Production bool
}

func implicitCORSWildcard(environment string) bool {
	switch env.NormalizeAppEnv(environment) {
	case "local", "development", "dev", "test", "testing":
		return true
	default:
		return false
	}
}

func corsDefaults(production, allowWildcard bool) CORSConfig {
	origins := []string(nil)
	if allowWildcard && !production {
		origins = []string{"*"}
	}
	return CORSConfig{
		AllowOrigins: origins,
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		AllowHeaders: "Content-Type, Authorization, X-Requested-With, X-CSRF-TOKEN, X-Idempotency-Key",
		MaxAge:       600,
		Production:   production,
	}
}

// DefaultCORSConfig returns development CORS defaults (permissive `*`).
// Kernel HTTP uses CORSFromEnv with the bootstrapped environment snapshot.
func DefaultCORSConfig() CORSConfig {
	return corsDefaults(false, true)
}

// CORSFromEnv builds CORS middleware from environment variables and the
// bootstrapped application environment (not a live APP_ENV parse).
func CORSFromEnv(environment string) routing.MiddlewareFunc {
	environment = env.NormalizeAppEnv(environment)
	production := environment == "production"
	cfg := corsDefaults(production, implicitCORSWildcard(environment))
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
	cfg.Production = production
	return CORSWith(cfg)
}

// CORSWith returns CORS middleware for the given config.
func CORSWith(cfg CORSConfig) routing.MiddlewareFunc {
	cfg = sanitizeCORSConfig(cfg)
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
				if cfg.AllowCredentials && allowOrigin != "" && allowOrigin != "*" {
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

// sanitizeCORSConfig strips insecure combinations (credentials+wildcard, production wildcard).
func sanitizeCORSConfig(cfg CORSConfig) CORSConfig {
	explicit := make([]string, 0, len(cfg.AllowOrigins))
	hasWildcard := false
	for _, o := range cfg.AllowOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			hasWildcard = true
			continue
		}
		explicit = append(explicit, o)
	}
	switch {
	case cfg.AllowCredentials || cfg.Production:
		cfg.AllowOrigins = explicit
	case hasWildcard:
		cfg.AllowOrigins = []string{"*"}
	default:
		cfg.AllowOrigins = explicit
	}
	return cfg
}

func resolveOrigin(allowed []string, requestOrigin string) string {
	if len(allowed) == 0 {
		return ""
	}
	for _, o := range allowed {
		if o == "*" {
			return "*"
		}
		if requestOrigin != "" && strings.EqualFold(o, requestOrigin) {
			return requestOrigin
		}
	}
	return ""
}
