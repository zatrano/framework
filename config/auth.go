package config

import "github.com/zatrano/framework/packages/env"

// Auth returns authentication guard and provider defaults.
func Auth() map[string]any {
	return map[string]any{
		"defaults": map[string]any{
			"guard":     env.Get("AUTH_GUARD", "web"),
			"provider":  env.Get("AUTH_PROVIDER", "users"),
			"passwords": env.Get("AUTH_PASSWORD_BROKER", "users"),
		},
		"guards": map[string]any{
			"web": map[string]any{
				"driver":   "session",
				"provider": "users",
			},
			// Session SPA/API cookie auth. Personal access tokens use packages/apitoken middleware.
			"api": map[string]any{
				"driver":   "session",
				"provider": "users",
			},
		},
		"providers": map[string]any{
			"users": map[string]any{
				"driver": "database",
				"table":  "users",
			},
		},
		"passwords": map[string]any{
			"users": map[string]any{
				"provider": "users",
				"table":    "password_reset_tokens",
				"expire":   60, // minutes
				"throttle": 60, // seconds
			},
		},
		"lockout": map[string]any{
			"max_attempts":  env.GetInt("AUTH_LOCKOUT_ATTEMPTS", 5),
			"decay_minutes": env.GetInt("AUTH_LOCKOUT_DECAY", 1),
		},
		"two_factor": map[string]any{
			"issuer":               env.Get("AUTH_2FA_ISSUER", env.Get("APP_NAME", "ZATRANO")),
			"remember_device_days": env.GetInt("AUTH_2FA_REMEMBER_DEVICE_DAYS", 30),
		},
	}
}
