package bootstrap

import (
	"fmt"
	"os"
	"strings"

	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/env"
)

// BootProfiles lists valid APP_BOOT values.
func BootProfiles() []string {
	return []string{"app", "api", "web", "minimal", "core", "kernel", "demo"}
}

// ResolveProfile normalizes an APP_BOOT value.
func ResolveProfile(name string) (string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "", "app", "default":
		return "app", nil
	case "api", "web", "minimal", "core", "demo":
		return name, nil
	case "kernel":
		return "core", nil
	default:
		return "", fmt.Errorf("unknown APP_BOOT %q (want: %s)", name, strings.Join(BootProfiles(), "|"))
	}
}

// Profile returns an application for the given boot profile name.
func Profile(name string) (*core.Application, error) {
	resolved, err := ResolveProfile(name)
	if err != nil {
		return nil, err
	}
	switch resolved {
	case "api":
		return APIApp(), nil
	case "web":
		return WebApp(), nil
	case "minimal":
		return MinimalApp(), nil
	case "core":
		return CoreApp(), nil
	case "demo":
		return DemoApp(), nil
	default:
		return App(), nil
	}
}

// MustProfile is Profile and panics on unknown names.
func MustProfile(name string) *core.Application {
	app, err := Profile(name)
	if err != nil {
		panic(err)
	}
	return app
}

// FromEnv selects a boot profile from APP_BOOT (after loading .env).
// defaultProfile is used when APP_BOOT is unset (defaults to "app").
func FromEnv(defaultProfile ...string) *core.Application {
	loadDotEnv()
	def := "app"
	if len(defaultProfile) > 0 && strings.TrimSpace(defaultProfile[0]) != "" {
		def = defaultProfile[0]
	}
	raw := env.Get("APP_BOOT", def)
	app, err := Profile(raw)
	if err != nil {
		panic(err)
	}
	return app
}

// CurrentBootProfile returns the configured APP_BOOT value (normalized), loading .env first.
func CurrentBootProfile(defaultProfile ...string) string {
	loadDotEnv()
	def := "app"
	if len(defaultProfile) > 0 && strings.TrimSpace(defaultProfile[0]) != "" {
		def = defaultProfile[0]
	}
	resolved, err := ResolveProfile(env.Get("APP_BOOT", def))
	if err != nil {
		return def
	}
	return resolved
}

func loadDotEnv() {
	candidates := []string{".env"}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, wd+string(os.PathSeparator)+".env")
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			_ = env.Load(path)
			return
		}
	}
}
