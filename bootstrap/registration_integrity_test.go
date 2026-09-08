package bootstrap

import (
	"testing"

	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel"
)

type bindProbe struct{ key string }

func (p *bindProbe) Register(app contracts.App) error {
	app.Container().Instance(p.key, true)
	return nil
}
func (p *bindProbe) Boot(contracts.App) error { return nil }

func registerIntegrityAddons(t *testing.T) {
	t.Helper()
	addons.ClearRegistry()
	clearEnablement()
	t.Cleanup(func() {
		clearEnablement()
		addons.ClearRegistry()
	})
	addons.Register(addons.Meta{
		Name: "keep", Key: "keep",
		Factory: func() contracts.Provider { return &bindProbe{key: "keep"} },
	})
	addons.Register(addons.Meta{
		Name: "skip", Key: "skip",
		Factory: func() contracts.Provider { return &bindProbe{key: "skip"} },
	})
}

func TestRegisterEnablementLastWriteWins(t *testing.T) {
	registerIntegrityAddons(t)
	RegisterEnablement([]string{"keep", "skip"})
	RegisterEnablement([]string{"keep"})
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := App()
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if !app.Bound("keep") {
		t.Fatal("last RegisterEnablement must win")
	}
	if app.Bound("skip") {
		t.Fatal("replaced enablement must not boot skip")
	}
}

func TestTwoAppsShareProcessGlobalRegistry(t *testing.T) {
	registerIntegrityAddons(t)
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	a := App(WithAddons("keep"))
	b := App(WithAddons("keep", "skip"))
	if err := a.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if err := b.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if !a.Bound("keep") || a.Bound("skip") {
		t.Fatal("app A WithAddons must select keep only")
	}
	if !b.Bound("keep") || !b.Bound("skip") {
		t.Fatal("app B WithAddons must select keep and skip")
	}
	names := addons.Names()
	if len(names) != 2 {
		t.Fatalf("process registry is shared, names=%v", names)
	}
}

func TestWithAddonsIsolatesApplicationSelection(t *testing.T) {
	registerIntegrityAddons(t)
	RegisterEnablement([]string{"keep", "skip"})
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := App(WithAddons("keep"))
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.Bound("skip") {
		t.Fatal("WithAddons isolates selection; it does not clone the registry")
	}
	if _, ok := addons.Lookup("skip"); !ok {
		t.Fatal("skip must remain in the process-global registry")
	}
}

func TestDuplicateAddonNamePanics(t *testing.T) {
	addons.ClearRegistry()
	t.Cleanup(addons.ClearRegistry)
	addons.Register(addons.Meta{Name: "dup"})
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate addon name must panic")
		}
	}()
	addons.Register(addons.Meta{Name: "dup"})
}

func TestAppDoesNotUsePerApplicationRegistry(t *testing.T) {
	registerIntegrityAddons(t)
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	_ = App(WithAddons("keep"))
	_ = App(WithAddons("skip"))
	if len(addons.Names()) != 2 {
		t.Fatal("two App() values must share one process-global imported set")
	}
	_ = kernel.NewApplication(t.TempDir())
}
