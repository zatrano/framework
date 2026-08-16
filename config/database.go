package config

import (
	"strings"

	"github.com/zatrano/framework/packages/database"
	"github.com/zatrano/framework/packages/env"
)

// Database returns database configuration.
//
// Env:
//
//	DB_CONNECTION=mysql                 # default connection name
//	DB_CONNECTIONS=mysql,pgsql,sqlite   # enabled connections (multi-DB); default = DB_CONNECTION only
//
// Shared fallbacks: DB_HOST, DB_PORT, DB_DATABASE, DB_USERNAME, DB_PASSWORD, DB_SSLMODE, DB_SERVICE
// Per-connection overrides: DB_<NAME>_HOST, DB_<NAME>_PORT, … (e.g. DB_MYSQL_HOST, DB_PGSQL_DATABASE)
func Database() map[string]any {
	defaultName := database.NormalizeDriverName(env.GetNonEmpty("DB_CONNECTION", "sqlite"))
	enabled := parseEnabledConnections(env.Get("DB_CONNECTIONS", ""), defaultName)

	connections := map[string]any{}
	for _, name := range enabled {
		connections[name] = connectionConfig(name, defaultName)
	}

	return map[string]any{
		"default":     defaultName,
		"connections": connections,
	}
}

func parseEnabledConnections(raw, defaultName string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{defaultName}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		n := database.NormalizeDriverName(p)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	if !seen[defaultName] {
		out = append([]string{defaultName}, out...)
	}
	if len(out) == 0 {
		return []string{defaultName}
	}
	return out
}

func connectionConfig(name, defaultName string) map[string]any {
	prefix := "DB_" + strings.ToUpper(name) + "_"
	shared := name == defaultName

	host := firstNonEmpty(env.Get(prefix+"HOST"), pick(shared, env.Get("DB_HOST", "127.0.0.1")))
	pass := firstNonEmpty(env.Get(prefix+"PASSWORD"), env.Get("DB_PASSWORD", ""))

	switch name {
	case "sqlite":
		db := firstNonEmpty(env.Get(prefix+"DATABASE"), pick(shared, env.GetNonEmpty("DB_DATABASE", "database/database.sqlite")))
		if !shared && db == "database/database.sqlite" {
			db = "database/" + name + ".sqlite"
		}
		return map[string]any{
			"driver":   "sqlite",
			"database": db,
		}
	case "mysql":
		return map[string]any{
			"driver":   "mysql",
			"host":     host,
			"port":     firstNonEmpty(env.Get(prefix+"PORT"), pick(shared, env.GetNonEmpty("DB_PORT", "3306")), "3306"),
			"database": firstNonEmpty(env.Get(prefix+"DATABASE"), pick(shared, env.GetNonEmpty("DB_DATABASE", "zatrano")), "zatrano"),
			"username": firstNonEmpty(env.Get(prefix+"USERNAME"), pick(shared, env.GetNonEmpty("DB_USERNAME", "root")), "root"),
			"password": pass,
			"charset":  firstNonEmpty(env.Get(prefix+"CHARSET"), env.GetNonEmpty("DB_CHARSET", "utf8mb4")),
		}
	case "pgsql":
		return map[string]any{
			"driver":   "pgsql",
			"host":     host,
			"port":     firstNonEmpty(env.Get(prefix+"PORT"), pick(shared, env.GetNonEmpty("DB_PORT", "5432")), "5432"),
			"database": firstNonEmpty(env.Get(prefix+"DATABASE"), pick(shared, env.GetNonEmpty("DB_DATABASE", "zatrano")), "zatrano"),
			"username": firstNonEmpty(env.Get(prefix+"USERNAME"), pick(shared, env.GetNonEmpty("DB_USERNAME", "postgres")), "postgres"),
			"password": pass,
			"sslmode":  firstNonEmpty(env.Get(prefix+"SSLMODE"), env.GetNonEmpty("DB_SSLMODE", "disable")),
		}
	case "mssql":
		return map[string]any{
			"driver":   "mssql",
			"host":     host,
			"port":     firstNonEmpty(env.Get(prefix+"PORT"), pick(shared, env.GetNonEmpty("DB_PORT", "1433")), "1433"),
			"database": firstNonEmpty(env.Get(prefix+"DATABASE"), pick(shared, env.GetNonEmpty("DB_DATABASE", "zatrano")), "zatrano"),
			"username": firstNonEmpty(env.Get(prefix+"USERNAME"), pick(shared, env.GetNonEmpty("DB_USERNAME", "sa")), "sa"),
			"password": pass,
		}
	case "oracle":
		return map[string]any{
			"driver":   "oracle",
			"host":     host,
			"port":     firstNonEmpty(env.Get(prefix+"PORT"), pick(shared, env.GetNonEmpty("DB_PORT", "1521")), "1521"),
			"database": firstNonEmpty(env.Get(prefix+"DATABASE"), pick(shared, env.GetNonEmpty("DB_DATABASE", "FREEPDB1")), "FREEPDB1"),
			"service":  firstNonEmpty(env.Get(prefix+"SERVICE"), env.GetNonEmpty("DB_SERVICE", ""), firstNonEmpty(env.Get(prefix+"DATABASE"), env.GetNonEmpty("DB_DATABASE", "FREEPDB1"))),
			"username": firstNonEmpty(env.Get(prefix+"USERNAME"), pick(shared, env.GetNonEmpty("DB_USERNAME", "system")), "system"),
			"password": pass,
		}
	default:
		return map[string]any{
			"driver":   name,
			"host":     host,
			"port":     firstNonEmpty(env.Get(prefix + "PORT")),
			"database": firstNonEmpty(env.Get(prefix+"DATABASE"), env.Get("DB_DATABASE", "")),
			"username": firstNonEmpty(env.Get(prefix+"USERNAME"), env.Get("DB_USERNAME", "")),
			"password": pass,
		}
	}
}

func pick(use bool, v string) string {
	if use {
		return v
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
