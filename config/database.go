package config

import "github.com/zatrano/framework/packages/env"

// Database returns database configuration.
func Database() map[string]any {
	return map[string]any{
		"default": env.Get("DB_CONNECTION", "sqlite"),
		"connections": map[string]any{
			"sqlite": map[string]any{
				"driver":   "sqlite",
				"database": env.Get("DB_DATABASE", "database/database.sqlite"),
			},
			"mysql": map[string]any{
				"driver":   "mysql",
				"host":     env.Get("DB_HOST", "127.0.0.1"),
				"port":     env.Get("DB_PORT", "3306"),
				"database": env.Get("DB_DATABASE", "zatrano"),
				"username": env.Get("DB_USERNAME", "root"),
				"password": env.Get("DB_PASSWORD", ""),
			},
			"pgsql": map[string]any{
				"driver":   "pgsql",
				"host":     env.Get("DB_HOST", "127.0.0.1"),
				"port":     env.Get("DB_PORT", "5432"),
				"database": env.Get("DB_DATABASE", "zatrano"),
				"username": env.Get("DB_USERNAME", "postgres"),
				"password": env.Get("DB_PASSWORD", ""),
			},
		},
	}
}
