package bootstrap_test

import (
	"os"
	"testing"

	"github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/bootstrap/addons"
)

func TestResolveProfile(t *testing.T) {
	cases := map[string]string{
		"":        "app",
		"APP":     "app",
		"default": "app",
		"api":     "app",
		"web":     "app",
		"minimal": "app",
		"core":    "app",
		"kernel":  "app",
	}
	for in, want := range cases {
		got, err := bootstrap.ResolveProfile(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got != want {
			t.Fatalf("%q => %q, want %q", in, got, want)
		}
	}
	if _, err := bootstrap.ResolveProfile("nope"); err == nil {
		t.Fatal("expected error for unknown profile")
	}
	if _, err := bootstrap.ResolveProfile("demo"); err == nil {
		t.Fatal("demo boot profile is removed")
	}
}

func TestProfileAPIDoesNotBindMongo(t *testing.T) {
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app, err := bootstrap.Profile("api")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.Bound("mongo") {
		t.Fatal("api profile should not bind mongo")
	}
	for _, name := range bootstrap.PresetAPI {
		m, ok := addons.Lookup(name)
		if !ok {
			t.Fatalf("missing %q", name)
		}
		if !app.Bound(m.Key) {
			t.Fatalf("api profile should bind %q", name)
		}
	}
}

func TestFromEnvHonorsAPP_BOOT(t *testing.T) {
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	t.Setenv("APP_BOOT", "minimal")
	app := bootstrap.FromEnv("app")
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	for _, m := range addons.Available() {
		if app.Bound(m.Key) {
			t.Fatalf("minimal FromEnv should not bind addon %q", m.Name)
		}
	}
	_ = os.Unsetenv("APP_BOOT")
}
