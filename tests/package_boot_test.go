package tests

import (
	"testing"

	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/session"

	"github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/bootstrap/addons"
)

func TestMinimalAppHasNoAddonsBound(t *testing.T) {
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := bootstrap.App(bootstrap.Minimal())
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	for _, m := range addons.Available() {
		if app.Bound(m.Key) {
			t.Fatalf("minimal app should not bind addon %q (%s)", m.Name, m.Key)
		}
	}
	if auth.From(app) == nil {
		t.Fatal("foundation auth should be available on MinimalApp")
	}
	if session.From(app) == nil {
		t.Fatal("foundation session should be available on MinimalApp")
	}
}

func TestCoreAppHasNoSession(t *testing.T) {
	app := bootstrap.App(bootstrap.Kernel())
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if session.From(app) != nil {
		t.Fatal("core app should not boot session")
	}
	if auth.From(app) != nil {
		t.Fatal("core app should not boot auth")
	}
	if app.Exceptions() == nil {
		t.Fatal("core app should boot exceptions")
	}
	if app.Encrypter() == nil {
		t.Fatal("core app should boot encrypter")
	}
}

func TestKernelOptionBoots(t *testing.T) {
	if bootstrap.App(bootstrap.Kernel()) == nil {
		t.Fatal("kernel option must return an app")
	}
}

func TestFullAppBindsEnabledAddons(t *testing.T) {
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := bootstrap.App()
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	enabled := map[string]bool{}
	for _, name := range bootstrap.EnabledAddons {
		enabled[name] = true
	}
	for _, m := range addons.Available() {
		if !enabled[m.Name] {
			if app.Bound(m.Key) {
				t.Fatalf("disabled addon %q should not bind %q", m.Name, m.Key)
			}
			continue
		}
		if !app.Bound(m.Key) {
			t.Fatalf("enabled addon %q should bind %q", m.Name, m.Key)
		}
	}
}

func TestAIAddonBinds(t *testing.T) {
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := bootstrap.App(bootstrap.WithAddons("ai"))
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	m, ok := addons.Lookup("ai")
	if !ok {
		t.Fatal("ai missing from addon registry")
	}
	if !app.Bound(m.Key) {
		t.Fatalf("ai addon should bind %q", m.Key)
	}
}

func TestAPIAppAndWebAppPresets(t *testing.T) {
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	for _, name := range bootstrap.PresetNames() {
		list, ok := bootstrap.Preset(name)
		if !ok {
			t.Fatalf("preset %q missing", name)
		}
		_ = list
	}

	api := bootstrap.App(bootstrap.WithPresetAPI())
	if err := api.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if api.Bound("mongo") {
		t.Fatal("APIApp should not bind heavy mongo by default")
	}

	web := bootstrap.App(bootstrap.WithPresetWeb())
	if err := web.Bootstrap(); err != nil {
		t.Fatal(err)
	}
}

func TestAddonConfigLoaded(t *testing.T) {
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	minimal := bootstrap.App(bootstrap.Minimal())
	if err := minimal.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if minimal.Config().Get("database") == nil {
		t.Fatal("foundation database config should load on MinimalApp")
	}
	if minimal.Config().Get("auth") == nil {
		t.Fatal("foundation auth config should load on MinimalApp")
	}
	if minimal.Config().Get("notifications") == nil {
		t.Fatal("foundation notifications config should load on MinimalApp")
	}
	if minimal.Config().Get("mongo") != nil || minimal.Config().Get("oauth") != nil {
		t.Fatal("MinimalApp must not load addon configs the kernel no longer owns")
	}

	app := bootstrap.App(bootstrap.WithAddons("ai"))
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.Config().Get("ai") == nil {
		t.Fatal("ai config namespace should load when the ai addon is enabled")
	}
}
