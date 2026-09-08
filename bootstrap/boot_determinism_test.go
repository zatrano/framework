package bootstrap

import (
	"testing"

	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/contracts"
)

type orderProbe struct {
	name  string
	order *[]string
}

func (p *orderProbe) Register(contracts.App) error {
	*p.order = append(*p.order, p.name)
	return nil
}

func (p *orderProbe) Boot(contracts.App) error { return nil }

func registerDeterminismAddons(t *testing.T, order *[]string) {
	t.Helper()
	addons.ClearRegistry()
	clearEnablement()
	t.Cleanup(func() {
		clearEnablement()
		addons.ClearRegistry()
	})
	addons.Register(addons.Meta{
		Name: "auth", Order: 50, Requires: []string{"hashing", "session"},
		Factory: func() contracts.Provider { return &orderProbe{name: "auth", order: order} },
	})
	addons.Register(addons.Meta{
		Name: "session", Order: 130,
		Factory: func() contracts.Provider { return &orderProbe{name: "session", order: order} },
	})
	addons.Register(addons.Meta{
		Name: "hashing", Order: 15,
		Factory: func() contracts.Provider { return &orderProbe{name: "hashing", order: order} },
	})
	addons.Register(addons.Meta{
		Name: "zeta", Order: 1,
		Factory: func() contracts.Provider { return &orderProbe{name: "zeta", order: order} },
	})
	addons.Register(addons.Meta{
		Name: "alpha", Order: 1,
		Factory: func() contracts.Provider { return &orderProbe{name: "alpha", order: order} },
	})
}

func TestAppProviderSlicePreservesOrderMetas(t *testing.T) {
	var order []string
	registerDeterminismAddons(t, &order)
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")

	metas, err := addons.Resolve("auth", "zeta", "alpha", "session", "hashing")
	if err != nil {
		t.Fatal(err)
	}
	want := make([]string, len(metas))
	for i, m := range metas {
		want[i] = m.Name
	}

	app := App(WithAddons("auth", "zeta", "alpha", "session", "hashing"))
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	if got := app.EnabledAddons(); !equalStringSlice(want, got) {
		t.Fatalf("EnabledAddons want %v got %v", want, got)
	}
	if !equalStringSlice(want, order) {
		t.Fatalf("provider Register order want %v got %v", want, order)
	}
}

func TestAppProviderSliceRepeatedIdentical(t *testing.T) {
	names := []string{"auth", "zeta", "alpha", "session", "hashing"}
	var first []string
	for i := 0; i < 8; i++ {
		var order []string
		registerDeterminismAddons(t, &order)
		t.Setenv("DB_CONNECTION", "")
		t.Setenv("DB_CONNECTIONS", "")
		app := App(WithAddons(names...))
		if err := app.Bootstrap(); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = append([]string(nil), order...)
			continue
		}
		if !equalStringSlice(first, order) {
			t.Fatalf("run %d: want %v got %v", i, first, order)
		}
		if !equalStringSlice(first, app.EnabledAddons()) {
			t.Fatalf("run %d EnabledAddons: want %v got %v", i, first, app.EnabledAddons())
		}
	}
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
