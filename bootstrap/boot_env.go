package bootstrap

import (
	"fmt"
	"os"
	"strings"

	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/kernel/env"
)

// BootProfiles lists valid APP_BOOT values.
func BootProfiles() []string {
	return []string{"app"}
}

// ResolveProfile normalizes an APP_BOOT value.
// Former api/web/minimal/core/kernel names still resolve; App() is always kernel
// plus whatever the process blank-imported.
func ResolveProfile(name string) (string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "", "app", "default", "api", "web", "minimal", "core", "kernel":
		return "app", nil
	default:
		return "", fmt.Errorf("unknown APP_BOOT %q (want: app)", name)
	}
}

// Profile returns an application for the given boot profile name.
func Profile(name string) (*kernel.Application, error) {
	if _, err := ResolveProfile(name); err != nil {
		return nil, err
	}
	return App(), nil
}

// MustProfile is Profile and panics on unknown names.
func MustProfile(name string) *kernel.Application {
	app, err := Profile(name)
	if err != nil {
		panic(err)
	}
	return app
}

// FromEnv selects a boot profile from APP_BOOT (after loading .env).
func FromEnv(defaultProfile ...string) *kernel.Application {
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
