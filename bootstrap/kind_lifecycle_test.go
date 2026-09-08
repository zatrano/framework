package bootstrap

import (
	"context"
	"testing"

	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/contracts"
)

type serviceLP struct {
	starts int
	stops  int
}

func (p *serviceLP) Register(app contracts.App) error {
	app.Container().Instance("svc-lp", true)
	return nil
}
func (p *serviceLP) Boot(contracts.App) error { return nil }
func (p *serviceLP) Start(contracts.App) error {
	p.starts++
	return nil
}
func (p *serviceLP) Stop(context.Context) error {
	p.stops++
	return nil
}

func TestServiceMayParticipateInLifecycle(t *testing.T) {
	addons.ClearRegistry()
	clearEnablement()
	t.Cleanup(func() {
		clearEnablement()
		addons.ClearRegistry()
	})
	lp := &serviceLP{}
	addons.Register(addons.Meta{
		Name: "features",
		Factory: func() contracts.Provider {
			return lp
		},
	})
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := App(WithAddons("features"))
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if !app.Bound("svc-lp") {
		t.Fatal("service provider must Register")
	}
	if lp.starts != 0 {
		t.Fatal("workers start only after Start")
	}
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if lp.starts != 1 {
		t.Fatalf("service may participate in lifecycle, starts=%d", lp.starts)
	}
	_ = app.Stop(context.Background())
	if lp.stops != 1 {
		t.Fatalf("stops=%d", lp.stops)
	}
}

func TestLibraryHasNoForcedLifecycleProvider(t *testing.T) {
	addons.ClearRegistry()
	clearEnablement()
	t.Cleanup(func() {
		clearEnablement()
		addons.ClearRegistry()
	})
	addons.Register(addons.Meta{
		Name: "collection",
		CLI: func(app contracts.App) []addons.CLICommand {
			return []addons.CLICommand{{Name: "collection:demo"}}
		},
	})
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := App(WithAddons("collection"))
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	names := app.EnabledAddons()
	if len(names) != 1 || names[0] != "collection" {
		t.Fatalf("library may be selected, names=%v", names)
	}
	if app.Bound("collection") {
		t.Fatal("factory-less library must not inject a Provider")
	}
	_ = app.Stop(context.Background())
}

func TestSharedModulePackagesHaveIndependentBootGraph(t *testing.T) {
	addons.ClearRegistry()
	clearEnablement()
	t.Cleanup(func() {
		clearEnablement()
		addons.ClearRegistry()
	})
	var order []string
	addons.Register(addons.Meta{
		Name: "session", Order: 130,
		Factory: func() contracts.Provider { return &orderProbe{name: "session", order: &order} },
	})
	addons.Register(addons.Meta{
		Name: "auth", Order: 50, Requires: []string{"session"},
		Factory: func() contracts.Provider { return &orderProbe{name: "auth", order: &order} },
	})
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := App(WithAddons("auth", "session"))
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "session" || order[1] != "auth" {
		t.Fatalf("shared-module packages boot independently: %v", order)
	}
}
