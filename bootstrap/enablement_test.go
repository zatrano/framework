package bootstrap

import (
	"testing"

	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel"
)

type g005Provider struct{ key string }

func (p *g005Provider) Register(app contracts.App) error {
	app.Container().Instance(p.key, true)
	return nil
}

func (p *g005Provider) Boot(contracts.App) error { return nil }

func registerG005Addons(t *testing.T) {
	t.Helper()
	addons.ClearRegistry()
	clearEnablement()
	t.Cleanup(func() {
		clearEnablement()
		addons.ClearRegistry()
	})
	addons.Register(addons.Meta{
		Name: "g005keep",
		Key:  "g005keep",
		Factory: func() contracts.Provider {
			return &g005Provider{key: "g005keep"}
		},
	})
	addons.Register(addons.Meta{
		Name: "g005skip",
		Key:  "g005skip",
		Factory: func() contracts.Provider {
			return &g005Provider{key: "g005skip"}
		},
	})
}

func bootApp(t *testing.T, opts ...Option) *kernel.Application {
	t.Helper()
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := App(opts...)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	return app
}

func TestAppWithoutManifestUsesDefaultMetas(t *testing.T) {
	registerG005Addons(t)

	app := bootApp(t)
	if !app.Bound("g005keep") || !app.Bound("g005skip") {
		t.Fatal("no manifest: App() must boot all imported addons (DefaultMetas)")
	}
}

func TestAppIgnoresFrameworkEnabledAddonsVar(t *testing.T) {
	registerG005Addons(t)

	prev := append([]string{}, EnabledAddons...)
	EnabledAddons = []string{"g005keep"}
	t.Cleanup(func() { EnabledAddons = prev })

	app := bootApp(t)
	if !app.Bound("g005skip") {
		t.Fatal("framework EnabledAddons must not drive App(); DefaultMetas boots all imported")
	}
}

func TestAppEmptyManifestIsKernelOnly(t *testing.T) {
	registerG005Addons(t)
	RegisterEnablement(nil)

	app := bootApp(t)
	if app.Bound("g005keep") || app.Bound("g005skip") {
		t.Fatal("registered empty manifest must not boot imported addons")
	}
}

func TestAppManifestIntersectsImported(t *testing.T) {
	registerG005Addons(t)
	RegisterEnablement([]string{"g005keep", "not-imported"})

	app := bootApp(t)
	if !app.Bound("g005keep") {
		t.Fatal("listed+imported addon must boot")
	}
	if app.Bound("g005skip") {
		t.Fatal("imported but not listed must not boot")
	}
}

func TestWithAddonsOverridesManifest(t *testing.T) {
	registerG005Addons(t)
	RegisterEnablement([]string{"g005keep", "g005skip"})

	app := bootApp(t, WithAddons("g005keep"))
	if !app.Bound("g005keep") {
		t.Fatal("WithAddons listed name must boot")
	}
	if app.Bound("g005skip") {
		t.Fatal("WithAddons is an override, not a merge with the manifest")
	}
}

func TestWithAddonsIntersectsImported(t *testing.T) {
	registerG005Addons(t)

	app := bootApp(t, WithAddons("g005keep", "not-imported"))
	if !app.Bound("g005keep") {
		t.Fatal("WithAddons ∩ Imported must boot the imported name")
	}
	if app.Bound("g005skip") {
		t.Fatal("unlisted imported addon must not boot")
	}
}

func TestInitDoesNotEnableByRegisteringAddon(t *testing.T) {
	registerG005Addons(t)
	RegisterEnablement([]string{"g005keep"})

	app := bootApp(t)
	if app.Bound("g005skip") {
		t.Fatal("addon init/registry must not enable; only the manifest (or WithAddons) does")
	}
}
