// Package env is the kernel environment primitive: process lookup and a
// small .env parser. Existing process variables always win over file values.
package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Load reads .env files and applies keys that are not already set in the
// process environment. With no paths it loads ".env" in the current directory.
// A missing file is an error; bootstrap callers ignore it.
func Load(paths ...string) error {
	if len(paths) == 0 {
		paths = []string{".env"}
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		values, err := Parse(data)
		if err != nil {
			return fmt.Errorf("env: %s: %w", path, err)
		}
		apply(values)
	}
	return nil
}

func apply(values map[string]string) {
	for key, value := range values {
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

// Lookup reports whether key is set in the process environment.
func Lookup(key string) (string, bool) {
	return os.LookupEnv(key)
}

// Set writes a process environment variable.
func Set(key, value string) error {
	return os.Setenv(key, value)
}

// Get returns an environment variable with an optional default.
func Get(key string, fallback ...string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

// NormalizeAppEnv trims and lowercases an application environment name.
// Bootstrap stores this once; security policy must not re-parse APP_ENV.
func NormalizeAppEnv(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// GetNonEmpty is like Get but treats blank/whitespace values as unset (uses fallback).
// Prefer this for credentials where `.env` may contain `DB_USERNAME=`.
func GetNonEmpty(key string, fallback ...string) string {
	if value, ok := os.LookupEnv(key); ok {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

// GetBool returns an environment variable as bool.
func GetBool(key string, fallback ...bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return false
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		if len(fallback) > 0 {
			return fallback[0]
		}
		return false
	}
}

// GetInt returns an environment variable as int.
func GetInt(key string, fallback ...int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	return parsed
}

// Sensitive reports whether key looks like a credential. Used so configuration
// errors can name the variable without echoing the value.
func Sensitive(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return false
	}
	for _, n := range []string{
		"password", "passwd", "secret", "token", "api_key", "apikey",
		"authorization", "credential", "private", "app_key", "dsn",
		"database_url", "db_url",
	} {
		if k == n || strings.Contains(k, n) {
			return true
		}
	}
	return false
}

// IntOr parses key as a base-10 integer. Unset or blank uses fallback.
// Invalid values return a configuration error that names the variable and
// expected type. Secret-like keys do not echo the received value.
func IntOr(key string, fallback int) (int, error) {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, ConfigError(key, "integer", strings.TrimSpace(value))
	}
	return n, nil
}

// ConfigError describes a configuration parse failure without leaking secrets.
func ConfigError(key, expected, received string) error {
	key = strings.TrimSpace(key)
	if Sensitive(key) {
		return fmt.Errorf("configuration error: %s: expected %s, received a value that could not be parsed", key, expected)
	}
	return fmt.Errorf("configuration error: %s: expected %s, received %q", key, expected, received)
}
