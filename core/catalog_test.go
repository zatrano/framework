package core

import "testing"

func TestCatalogCoversLayers(t *testing.T) {
	if len(PackagesByLayer(LayerKernel)) < 5 {
		t.Fatalf("expected kernel packages, got %d", len(PackagesByLayer(LayerKernel)))
	}
	if len(PackagesByLayer(LayerFoundation)) < 10 {
		t.Fatalf("expected foundation packages, got %d", len(PackagesByLayer(LayerFoundation)))
	}
	if len(PackagesByLayer(LayerAddon)) < 20 {
		t.Fatalf("expected addon packages, got %d", len(PackagesByLayer(LayerAddon)))
	}
}

func TestCatalogAddonKinds(t *testing.T) {
	services := 0
	libraries := 0
	for _, p := range PackagesByLayer(LayerAddon) {
		switch p.EffectiveKind() {
		case KindService:
			services++
		case KindLibrary:
			libraries++
		default:
			t.Fatalf("addon %q has unexpected kind %q", p.Name, p.EffectiveKind())
		}
		if p.Description == "" {
			t.Fatalf("addon %q missing description", p.Name)
		}
	}
	if services < 10 {
		t.Fatalf("expected service addons, got %d", services)
	}
	if libraries < 10 {
		t.Fatalf("expected library addons, got %d", libraries)
	}
	mongo, ok := LookupPackage("mongo")
	if !ok || mongo.EffectiveKind() != KindService {
		t.Fatal("mongo should be a service addon")
	}
	collection, ok := LookupPackage("collection")
	if !ok || collection.EffectiveKind() != KindLibrary {
		t.Fatal("collection should be a library addon")
	}
	oauth, ok := LookupPackage("oauth")
	if !ok || oauth.EffectiveKind() != KindService {
		t.Fatal("oauth should be a service addon")
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
