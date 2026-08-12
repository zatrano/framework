package env

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Load reads environment variables from the given .env files.
func Load(paths ...string) error {
	if len(paths) == 0 {
		paths = []string{".env"}
	}
	return godotenv.Load(paths...)
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
