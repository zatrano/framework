package kernel

import "testing"

func TestCatalogCoversLayers(t *testing.T) {
	if len(PackagesByLayer(LayerPrimitive)) < 5 {
		t.Fatalf("expected primitive packages, got %d", len(PackagesByLayer(LayerPrimitive)))
	}
	if len(PackagesByLayer(LayerFoundation)) < 10 {
		t.Fatalf("expected foundation packages, got %d", len(PackagesByLayer(LayerFoundation)))
	}
	if len(PackagesByLayer(LayerIntelligence)) != 3 {
		t.Fatalf("expected 3 intelligence packages, got %d", len(PackagesByLayer(LayerIntelligence)))
	}
	if len(PackagesByLayer(LayerAddon)) < 20 {
		t.Fatalf("expected addon packages, got %d", len(PackagesByLayer(LayerAddon)))
	}
	ai, ok := LookupPackage("ai")
	if !ok || ai.Layer != LayerIntelligence || ai.EffectiveKind() != KindService {
		t.Fatal("ai should be an intelligence service")
	}
	rag, ok := LookupPackage("rag")
	if !ok || rag.Layer != LayerIntelligence || rag.EffectiveKind() != KindLibrary {
		t.Fatal("rag should be an intelligence library")
	}
	agent, ok := LookupPackage("agent")
	if !ok || agent.Layer != LayerIntelligence || agent.EffectiveKind() != KindLibrary {
		t.Fatal("agent should be an intelligence library")
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
