package kernel

import "testing"

func TestCatalogPrimitivesOnly(t *testing.T) {
	if len(PackagesByLayer(LayerPrimitive)) < 5 {
		t.Fatalf("expected primitive packages, got %d", len(PackagesByLayer(LayerPrimitive)))
	}
	if len(PackagesByLayer(LayerFoundation)) != 0 {
		t.Fatalf("kernel catalog must not list foundation packages, got %d", len(PackagesByLayer(LayerFoundation)))
	}
	if len(PackagesByLayer(LayerIntelligence)) != 0 {
		t.Fatalf("kernel catalog must not list intelligence packages, got %d", len(PackagesByLayer(LayerIntelligence)))
	}
	if len(PackagesByLayer(LayerAddon)) != 0 {
		t.Fatalf("kernel catalog must not list addon packages, got %d", len(PackagesByLayer(LayerAddon)))
	}
	for _, p := range Catalog {
		if p.Layer != LayerPrimitive {
			t.Errorf("%s layer=%s want primitive", p.Name, p.Layer)
		}
	}
	for _, name := range []string{"auth", "database", "ai", "agent", "billing"} {
		if _, ok := LookupPackage(name); ok {
			t.Errorf("%s must not be in the kernel catalog", name)
		}
	}
}

func TestCatalogKernelInternals(t *testing.T) {
	for _, name := range []string{"cookie", "support"} {
		p, ok := LookupPackage(name)
		if !ok {
			t.Fatalf("%s missing from catalog", name)
		}
		if p.Layer != LayerPrimitive {
			t.Fatalf("%s layer=%q want primitive", name, p.Layer)
		}
	}
	for _, name := range []string{"safepath", "layout"} {
		if _, ok := LookupPackage(name); ok {
			t.Fatalf("%s should stay out of the catalog (kernel-internal helper)", name)
		}
	}
}

func TestMakeMissingService(t *testing.T) {
	app := NewApplication(".")
	if _, err := app.Make("missing"); err == nil {
		t.Fatal("expected error for missing binding")
	}
	if app.Bound("mongo") {
		t.Fatal("mongo must not be bound without addon providers")
	}
}
