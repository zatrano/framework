package config

import "github.com/zatrano/framework/env"

// Session returns session configuration.
func Session() map[string]any {
	return map[string]any{
		"driver":   env.Get("SESSION_DRIVER", "file"),
		"lifetime": env.GetInt("SESSION_LIFETIME", 120),
		"path":     "storage/framework/sessions",
		"cookie":   "zatrano_session",
	}
}
