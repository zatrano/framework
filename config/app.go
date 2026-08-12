package config

import "github.com/zatrano/framework/packages/env"

// App returns application configuration.
func App() map[string]any {
	return map[string]any{
		"name":     env.Get("APP_NAME", "ZATRANO"),
		"env":      env.Get("APP_ENV", "local"),
		"debug":    env.GetBool("APP_DEBUG", true),
		"url":      env.Get("APP_URL", "http://localhost:8080"),
		"port":     env.Get("APP_PORT", "8080"),
		"key":      env.Get("APP_KEY"),
		"locale":   env.Get("APP_LOCALE", "en"),
		"fallback": env.Get("APP_FALLBACK_LOCALE", "en"),
		"cors": map[string]any{
			"enabled":           env.GetBool("CORS_ENABLED", true),
			"allowed_origins":   env.Get("CORS_ALLOWED_ORIGINS", "*"),
			"allowed_methods":   env.Get("CORS_ALLOWED_METHODS", "GET, POST, PUT, PATCH, DELETE, OPTIONS"),
			"allowed_headers":   env.Get("CORS_ALLOWED_HEADERS", "Content-Type, Authorization, X-Requested-With, X-CSRF-TOKEN, X-Idempotency-Key"),
			"allow_credentials": env.GetBool("CORS_ALLOW_CREDENTIALS", false),
			"max_age":           env.Get("CORS_MAX_AGE", "600"),
		},
	}
}
