package config

import "github.com/zatrano/framework/packages/env"

// Database returns database configuration for sqlite, mysql, pgsql, and mssql.
func Database() map[string]any {
	return map[string]any{
		"default": env.Get("DB_CONNECTION", "sqlite"),
		"connections": map[string]any{
			"sqlite": map[string]any{
				"driver":   "sqlite",
				"database": env.GetNonEmpty("DB_DATABASE", "database/database.sqlite"),
			},
			"mysql": map[string]any{
				"driver":   "mysql",
				"host":     env.Get("DB_HOST", "127.0.0.1"),
				"port":     env.GetNonEmpty("DB_PORT", "3306"),
				"database": env.GetNonEmpty("DB_DATABASE", "zatrano"),
				"username": env.GetNonEmpty("DB_USERNAME", "root"),
				"password": env.Get("DB_PASSWORD", ""),
				"charset":  env.GetNonEmpty("DB_CHARSET", "utf8mb4"),
			},
			"pgsql": map[string]any{
				"driver":   "pgsql",
				"host":     env.Get("DB_HOST", "127.0.0.1"),
				"port":     env.GetNonEmpty("DB_PORT", "5432"),
				"database": env.GetNonEmpty("DB_DATABASE", "zatrano"),
				"username": env.GetNonEmpty("DB_USERNAME", "postgres"),
				"password": env.Get("DB_PASSWORD", ""),
				"sslmode":  env.GetNonEmpty("DB_SSLMODE", "disable"),
			},
			"mssql": map[string]any{
				"driver":   "mssql",
				"host":     env.Get("DB_HOST", "127.0.0.1"),
				"port":     env.GetNonEmpty("DB_PORT", "1433"),
				"database": env.GetNonEmpty("DB_DATABASE", "zatrano"),
				"username": env.GetNonEmpty("DB_USERNAME", "sa"),
				"password": env.Get("DB_PASSWORD", ""),
			},
		},
	}
}
