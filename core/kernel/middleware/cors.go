package middleware

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zatrano/framework/v3/core/kernel/env"
	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
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
// Bootstrap calls ValidateCORSEnv first. CORSWith still strips a wildcard
// when credentials are set, so a direct call cannot emit both.
func CORSFromEnv(environment string) routing.MiddlewareFunc {
	return CORSWith(corsConfigFromEnv(environment))
}

// ValidateCORSEnv reports a boot error when credentials are combined with a
// wildcard origin, including the implicit development wildcard.
func ValidateCORSEnv(environment string) error {
	cfg := corsConfigFromEnv(environment)
	if !cfg.AllowCredentials {
		return nil
	}
	for _, o := range cfg.AllowOrigins {
		if strings.TrimSpace(o) == "*" {
			return fmt.Errorf("cors: CORS_ALLOW_CREDENTIALS cannot be combined with a wildcard origin")
		}
	}
	return nil
}

func corsConfigFromEnv(environment string) CORSConfig {
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
	return cfg
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

	originsConfigured := len(cfg.AllowOrigins) > 0
	wildcard := false
	for _, o := range cfg.AllowOrigins {
		if o == "*" {
			wildcard = true
			break
		}
	}

	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if !originsConfigured {
				return next(req)
			}
			values := req.HeaderValues("Origin")
			allowOrigin, matched := matchOrigin(cfg.AllowOrigins, values)
			varyOrigin := !wildcard && originPresent(values)

			writeAllow := func(resp *http.Response, headerValue string, headerSet bool) {
				if allowOrigin != "" {
					resp.Header("Access-Control-Allow-Origin", allowOrigin)
				}
				if cfg.AllowMethods != "" {
					resp.Header("Access-Control-Allow-Methods", cfg.AllowMethods)
				}
				if headerSet {
					resp.Header("Access-Control-Allow-Headers", headerValue)
				}
				if cfg.ExposeHeaders != "" {
					resp.Header("Access-Control-Expose-Headers", cfg.ExposeHeaders)
				}
				if cfg.AllowCredentials && allowOrigin != "" && allowOrigin != "*" {
					resp.Header("Access-Control-Allow-Credentials", "true")
				}
				if cfg.MaxAge > 0 {
					resp.Header("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}
			}

			if req.Method() == "OPTIONS" && originPresent(values) && req.Header("Access-Control-Request-Method") != "" {
				if !matched {
					resp := http.Abort(404, "Not Found")
					if varyOrigin {
						resp.Header("Vary", "Origin")
					}
					return resp
				}
				resp := http.NoContent()
				hv, hs := allowHeadersValue(cfg.AllowHeaders, req.Header("Access-Control-Request-Headers"))
				writeAllow(resp, hv, hs)
				if wildcard {
					return resp
				}
				resp.Header("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
				return resp
			}

			resp := next(req)
			if resp == nil || !matched && !varyOrigin {
				return resp
			}
			if matched {
				writeAllow(resp, cfg.AllowHeaders, cfg.AllowHeaders != "")
			}
			if varyOrigin {
				setVaryOrigin(resp)
			}
			return resp
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

func originPresent(values []string) bool {
	for _, v := range values {
		if v != "" {
			return true
		}
	}
	return false
}

// matchOrigin reflects an origin only on an exact match of a single Origin
// value. Scheme, host, port, and a trailing slash are significant.
// A wildcard allow-list reflects "*" for any request, including one with no Origin.
func matchOrigin(allowed, values []string) (string, bool) {
	for _, o := range allowed {
		if o == "*" {
			return "*", true
		}
	}
	if len(values) != 1 || values[0] == "" {
		return "", false
	}
	for _, o := range allowed {
		if o == values[0] {
			return values[0], true
		}
	}
	return "", false
}

func setVaryOrigin(resp *http.Response) {
	existing := resp.Headers().Get("Vary")
	if existing == "" {
		resp.Header("Vary", "Origin")
		return
	}
	for _, part := range strings.Split(existing, ",") {
		if strings.EqualFold(strings.TrimSpace(part), "Origin") {
			return
		}
	}
	resp.Header("Vary", existing+", Origin")
}

// allowHeadersValue intersects a preflight Access-Control-Request-Headers
// list with the configured allow list. Names are compared case-insensitively
// and emitted with the configured spelling. An absent request header keeps
// the configured list. An empty intersection sets nothing.
func allowHeadersValue(configured, requested string) (string, bool) {
	asked := splitCSV(requested)
	if len(asked) == 0 {
		if strings.TrimSpace(configured) == "" {
			return "", false
		}
		return configured, true
	}
	allowed := splitCSV(configured)
	hit := make([]string, 0, len(asked))
	seen := make(map[string]struct{}, len(allowed))
	for _, req := range asked {
		for _, a := range allowed {
			if !strings.EqualFold(a, req) {
				continue
			}
			key := strings.ToLower(a)
			if _, ok := seen[key]; ok {
				break
			}
			seen[key] = struct{}{}
			hit = append(hit, a)
			break
		}
	}
	if len(hit) == 0 {
		return "", false
	}
	return strings.Join(hit, ", "), true
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
