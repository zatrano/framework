package database

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// BuildDSNFor builds a database/sql driver name and DSN without requiring the
// third-party driver to be linked. Opening still needs a blank-import of the
// matching packages/database/driver/<name> module (via db:setup).
func BuildDSNFor(cfg ConnectionConfig, basePath string) (driverName, dsn string, err error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Driver)) {
	case "sqlite", "sqlite3":
		path, err := ResolveSQLitePath(cfg.Database, basePath)
		if err != nil {
			return "", "", err
		}
		return "sqlite", path, nil
	case "mysql":
		charset := cfg.Charset
		if charset == "" {
			charset = "utf8mb4"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=true&loc=Local",
			cfg.Username, cfg.Password, cfg.Host, DefaultPort(cfg.Port, "3306"), cfg.Database, charset)
		return "mysql", dsn, nil
	case "pgsql", "postgres", "postgresql":
		ssl := cfg.SSLMode
		if ssl == "" {
			ssl = "disable"
		}
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, DefaultPort(cfg.Port, "5432"), cfg.Username, cfg.Password, cfg.Database, ssl)
		return "postgres", dsn, nil
	case "mssql", "sqlserver":
		query := url.Values{}
		query.Set("database", cfg.Database)
		u := &url.URL{
			Scheme:   "sqlserver",
			User:     url.UserPassword(cfg.Username, cfg.Password),
			Host:     fmt.Sprintf("%s:%s", cfg.Host, DefaultPort(cfg.Port, "1433")),
			RawQuery: query.Encode(),
		}
		return "sqlserver", u.String(), nil
	case "oracle", "ora":
		service := cfg.Service
		if service == "" {
			service = cfg.Database
		}
		if service == "" {
			return "", "", fmt.Errorf("oracle service/database name required")
		}
		u := url.URL{
			Scheme: "oracle",
			User:   url.UserPassword(cfg.Username, cfg.Password),
			Host:   fmt.Sprintf("%s:%s", cfg.Host, DefaultPort(cfg.Port, "1521")),
			Path:   "/" + service,
		}
		return "oracle", u.String(), nil
	default:
		return "", "", fmt.Errorf("unsupported database driver [%s]", cfg.Driver)
	}
}

// ResolveSQLitePath expands a relative sqlite path under basePath and ensures the file exists.
func ResolveSQLitePath(database, basePath string) (string, error) {
	path := database
	if path == "" {
		path = "database/database.sqlite"
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(basePath, path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, createErr := os.Create(path)
		if createErr != nil {
			return "", createErr
		}
		_ = f.Close()
	}
	return path, nil
}

// DefaultPort returns port or fallback.
func DefaultPort(port, fallback string) string {
	if strings.TrimSpace(port) == "" {
		return fallback
	}
	return port
}

// KnownDrivers lists first-party optional SQL drivers (install via db:setup).
func KnownDrivers() []string {
	return []string{"sqlite", "mysql", "pgsql", "mssql", "oracle"}
}

// DriverModulePath returns the Go module path for a first-party driver.
func DriverModulePath(name string) string {
	switch strings.ToLower(name) {
	case "sqlite", "sqlite3":
		return "github.com/zatrano/framework/packages/database/driver/sqlite"
	case "mysql":
		return "github.com/zatrano/framework/packages/database/driver/mysql"
	case "pgsql", "postgres", "postgresql":
		return "github.com/zatrano/framework/packages/database/driver/pgsql"
	case "mssql", "sqlserver":
		return "github.com/zatrano/framework/packages/database/driver/mssql"
	case "oracle", "ora":
		return "github.com/zatrano/framework/packages/database/driver/oracle"
	default:
		return ""
	}
}

// NormalizeDriverName maps aliases to canonical connection names.
func NormalizeDriverName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "sqlite", "sqlite3":
		return "sqlite"
	case "mysql":
		return "mysql"
	case "pgsql", "postgres", "postgresql":
		return "pgsql"
	case "mssql", "sqlserver":
		return "mssql"
	case "oracle", "ora":
		return "oracle"
	default:
		return strings.ToLower(strings.TrimSpace(name))
	}
}
