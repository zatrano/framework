package env_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/kernel/env"
)

func TestLoadDoesNotOverrideProcessEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("ZATRANO_ENV_LOAD_A=from-file\nZATRANO_ENV_LOAD_B=from-file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZATRANO_ENV_LOAD_A", "from-os")
	_ = os.Unsetenv("ZATRANO_ENV_LOAD_B")
	t.Cleanup(func() {
		_ = os.Unsetenv("ZATRANO_ENV_LOAD_B")
	})
	if err := env.Load(path); err != nil {
		t.Fatal(err)
	}
	if env.Get("ZATRANO_ENV_LOAD_A") != "from-os" {
		t.Fatalf("process env must win, got %q", env.Get("ZATRANO_ENV_LOAD_A"))
	}
	if env.Get("ZATRANO_ENV_LOAD_B") != "from-file" {
		t.Fatalf("missing keys come from file, got %q", env.Get("ZATRANO_ENV_LOAD_B"))
	}
}

func TestLoadMissingFile(t *testing.T) {
	if err := env.Load(filepath.Join(t.TempDir(), "nope.env")); err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestLookupAndSet(t *testing.T) {
	key := "ZATRANO_ENV_LOOKUP"
	_ = os.Unsetenv(key)
	if _, ok := env.Lookup(key); ok {
		t.Fatal("expected unset")
	}
	if err := env.Set(key, "x"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Unsetenv(key) })
	if v, ok := env.Lookup(key); !ok || v != "x" {
		t.Fatalf("got %q %v", v, ok)
	}
}

func TestGetBoolGetInt(t *testing.T) {
	t.Setenv("ZATRANO_ENV_BOOL", "yes")
	if !env.GetBool("ZATRANO_ENV_BOOL") {
		t.Fatal("bool")
	}
	t.Setenv("ZATRANO_ENV_INT", "42")
	if env.GetInt("ZATRANO_ENV_INT") != 42 {
		t.Fatal("int")
	}
	if env.GetInt("ZATRANO_ENV_INT_MISSING", 7) != 7 {
		t.Fatal("int fallback")
	}
}
