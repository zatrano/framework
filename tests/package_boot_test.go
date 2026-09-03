package tests

import (
	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/session"
	"testing"

	"github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/packages/mongo"
)

func TestMinimalAppHasNoAddonsBound(t *testing.T) {
	app := bootstrap.App(bootstrap.Minimal())
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	for _, m := range addons.Available() {
		if app.Bound(m.Key) {
			t.Fatalf("minimal app should not bind addon %q (%s)", m.Name, m.Key)
		}
	}
	if mongo.From(app) != nil {
		t.Fatal("mongo.From should be nil on MinimalApp")
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
	if len(bootstrap.EnabledAddons) == 0 && mongo.From(app) != nil {
		t.Fatal("mongo.From should be nil when EnabledAddons is empty")
	}
}

func TestDemoAppBindsDemoAddons(t *testing.T) {
	app := bootstrap.App(bootstrap.WithDemo())
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	for _, name := range bootstrap.DemoAddons {
		m, ok := addons.Lookup(name)
		if !ok {
			t.Fatalf("demo addon %q missing from registry", name)
		}
		if !app.Bound(m.Key) {
			t.Fatalf("demo addon %q should bind %q", m.Name, m.Key)
		}
	}
	if mongo.From(app) == nil {
		t.Fatal("mongo.From should resolve on DemoApp")
	}
}

func TestAPIAppAndWebAppPresets(t *testing.T) {
	for _, name := range bootstrap.PresetNames() {
		list, ok := bootstrap.Preset(name)
		if !ok || len(list) == 0 {
			t.Fatalf("preset %q missing", name)
		}
		for _, pkg := range list {
			if _, ok := addons.Lookup(pkg); !ok {
				t.Fatalf("preset %q references unknown addon %q", name, pkg)
			}
		}
	}

	api := bootstrap.App(bootstrap.WithPresetAPI())
	if err := api.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	for _, name := range bootstrap.PresetAPI {
		m, _ := addons.Lookup(name)
		if !api.Bound(m.Key) {
			t.Fatalf("APIApp should bind %q", name)
		}
	}
	if api.Bound("mongo") {
		t.Fatal("APIApp should not bind heavy mongo by default")
	}

	web := bootstrap.App(bootstrap.WithPresetWeb())
	if err := web.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	for _, name := range bootstrap.PresetWeb {
		m, _ := addons.Lookup(name)
		if !web.Bound(m.Key) {
			t.Fatalf("WebApp should bind %q", name)
		}
	}
}

func TestAddonConfigLoaded(t *testing.T) {
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

	app := bootstrap.App(bootstrap.WithDemo())
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if got := app.Config().GetString("mongo.uri", ""); got == "" {
		t.Fatal("mongo.uri config should be loaded")
	}
	if app.Config().Get("oauth") == nil {
		t.Fatal("oauth config namespace should be loaded")
	}
	if app.Config().Get("webauthn") == nil {
		t.Fatal("webauthn config namespace should be loaded")
	}
	if app.Config().GetString("billing.stripe_secret", "missing") == "missing" && app.Config().Get("billing") == nil {
		t.Fatal("billing config namespace should be loaded")
	}
}
