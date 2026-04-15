package config

import "strings"

// SanitizedSnapshot returns a nested map safe to print (secrets masked). Stable keys for scripting.
func SanitizedSnapshot(c *Config) map[string]any {
	out := map[string]any{
		"env":               c.Env,
		"app_name":          c.AppName,
		"http_addr":         c.HTTPAddr,
		"http_read_timeout": c.HTTPReadTimeout.String(),
		"database_url":      MaskConnectionURL(c.DatabaseURL),
		"database_required": c.DatabaseRequired,
		"redis_url":         MaskConnectionURL(c.RedisURL),
		"redis_required":    c.RedisRequired,
		"log_level":         c.LogLevel,
		"log_development":   c.LogDevelopment,
		"static_path":       c.StaticPath,
		"static_url_prefix": c.StaticURLPrefix,
		"migrations_dir":    c.MigrationsDir,
		"seeds_dir":         c.SeedsDir,
		"openapi_path":      c.OpenAPIPath,
		"http": map[string]any{
			"cors_enabled":           c.HTTP.CORSEnabled,
			"cors_allow_origins":     c.HTTP.CORSAllowOrigins,
			"cors_allow_credentials": c.HTTP.CORSAllowCredentials,
			"rate_limit_enabled":     c.HTTP.RateLimitEnabled,
			"rate_limit_max":         c.HTTP.RateLimitMax,
			"rate_limit_window":      c.HTTP.RateLimitWindow.String(),
			"rate_limit_redis":       c.HTTP.RateLimitRedis,
			"request_timeout":        c.HTTP.RequestTimeout.String(),
			"body_limit":             c.HTTP.BodyLimit,
		},
		"security": map[string]any{
			"session_enabled":     c.Security.SessionEnabled,
			"csrf_enabled":        c.Security.CSRFEnabled,
			"csrf_skip_prefixes":  c.Security.CSRFSkipPrefixes,
			"trusted_origins":     c.Security.TrustedOrigins,
			"jwt_secret":          MaskSecret(c.Security.JWTSecret),
			"jwt_issuer":          c.Security.JWTIssuer,
			"jwt_expiry":          c.Security.JWTExpiry.String(),
			"cookie_secure":       c.Security.CookieSecure,
			"demo_token_endpoint": c.Security.DemoTokenEndpoint,
		},
		"oauth": oauthSnapshot(&c.OAuth),
		"i18n": map[string]any{
			"enabled":           c.I18n.Enabled,
			"default_locale":    c.I18n.DefaultLocale,
			"supported_locales": c.I18n.SupportedLocales,
			"locales_dir":       c.I18n.LocalesDir,
			"cookie_name":       c.I18n.CookieName,
			"query_key":         c.I18n.QueryKey,
		},
	}
	return out
}

func oauthSnapshot(o *OAuth) map[string]any {
	prov := map[string]any{
		"google": map[string]any{
			"client_id":     strings.TrimSpace(o.Providers.Google.ClientID),
			"client_secret": MaskSecret(o.Providers.Google.ClientSecret),
			"scopes":        o.Providers.Google.Scopes,
		},
		"github": map[string]any{
			"client_id":     strings.TrimSpace(o.Providers.Github.ClientID),
			"client_secret": MaskSecret(o.Providers.Github.ClientSecret),
			"scopes":        o.Providers.Github.Scopes,
		},
	}
	return map[string]any{
		"enabled":   o.Enabled,
		"base_url":  strings.TrimSpace(o.BaseURL),
		"providers": prov,
	}
}

