package kernel_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
	"github.com/zatrano/framework/v2/kernel/config"
)

func configCacheFile(dir string) string {
	return filepath.Join(dir, "storage", "framework", "cache", "config.json")
}

func writeValidConfigCache(t *testing.T, dir string, appName string) {
	t.Helper()
	path := configCacheFile(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	repo := config.New()
	repo.Load("app", map[string]any{"name": appName, "env": "local", "debug": false})
	if err := config.SaveCache(path, repo); err != nil {
		t.Fatal(err)
	}
}

func TestConfigCacheMissingUsesEnv(t *testing.T) {
	t.Setenv("APP_NAME", "from-env")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &countingProvider{}
	app.RegisterProviders(p)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.Config().GetString("app.name") != "from-env" {
		t.Fatalf("got %q", app.Config().GetString("app.name"))
	}
	if p.registers != 1 || p.boots != 1 {
		t.Fatalf("register=%d boot=%d", p.registers, p.boots)
	}
}

func TestConfigCacheDisabledUsesEnv(t *testing.T) {
	dir := t.TempDir()
	writeValidConfigCache(t, dir, "from-cache")
	t.Setenv("APP_CONFIG_CACHE", "false")
	t.Setenv("APP_NAME", "from-env")
	app := kernel.NewApplication(dir)
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.Config().GetString("app.name") != "from-env" {
		t.Fatalf("disabled cache must use ENV, got %q", app.Config().GetString("app.name"))
	}
}

func TestConfigCacheValidWinsOverEnv(t *testing.T) {
	dir := t.TempDir()
	writeValidConfigCache(t, dir, "from-cache")
	t.Setenv("APP_NAME", "from-env")
	app := kernel.NewApplication(dir)
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.Config().GetString("app.name") != "from-cache" {
		t.Fatalf("valid cache must win, got %q", app.Config().GetString("app.name"))
	}
}

func TestCorruptConfigCacheFallsBackToEnv(t *testing.T) {
	dir := t.TempDir()
	path := configCacheFile(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_NAME", "from-env")
	app := kernel.NewApplication(dir)
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &countingProvider{}
	app.RegisterProviders(p)
	if err := app.Bootstrap(); err != nil {
		t.Fatal("corrupt cache must not fail bootstrap")
	}
	name := app.Config().GetString("app.name")
	if name == "" {
		t.Fatal("corrupt cache must not leave empty app config")
	}
	if name != "from-env" {
		t.Fatalf("corrupt cache is a cache miss, got %q", name)
	}
	if p.registers != 1 || p.boots != 1 {
		t.Fatalf("providers must still boot, register=%d boot=%d", p.registers, p.boots)
	}
}

func TestUnreadableConfigCacheFallsBackToEnv(t *testing.T) {
	dir := t.TempDir()
	path := configCacheFile(dir)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_NAME", "from-env")
	app := kernel.NewApplication(dir)
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &countingProvider{}
	app.RegisterProviders(p)
	if err := app.Bootstrap(); err != nil {
		t.Fatal("unreadable cache must not fail bootstrap")
	}
	if app.Config().GetString("app.name") != "from-env" {
		t.Fatalf("unreadable cache is a cache miss, got %q", app.Config().GetString("app.name"))
	}
	if p.registers != 1 || p.boots != 1 {
		t.Fatalf("providers must still boot, register=%d boot=%d", p.registers, p.boots)
	}
}

func TestCorruptConfigCacheStillEnforcesProductionAPPKEY(t *testing.T) {
	dir := t.TempDir()
	path := configCacheFile(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_KEY", "password")
	app := kernel.NewApplication(dir)
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err == nil {
		t.Fatal("production APP_KEY gate must still fail-closed")
	}
}

func TestCorruptConfigCacheProductionAcceptsEnvKey(t *testing.T) {
	dir := t.TempDir()
	path := configCacheFile(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	t.Setenv("APP_NAME", "from-env")
	app := kernel.NewApplication(dir)
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.Config().GetString("app.name") != "from-env" {
		t.Fatalf("got %q", app.Config().GetString("app.name"))
	}
	if app.Config().GetString("app.key") != strings.Repeat("s", 32) {
		t.Fatal("ENV APP_KEY must populate config after cache miss")
	}
}
