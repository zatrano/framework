package console

import (
	"testing"

	"github.com/zatrano/framework/kernel"
)

func TestEcosystemCatalogCoversLayers(t *testing.T) {
	if len(catalogByLayer(kernel.LayerPrimitive)) < 5 {
		t.Fatalf("expected primitives in catalogAll, got %d", len(catalogByLayer(kernel.LayerPrimitive)))
	}
	if len(catalogByLayer(kernel.LayerFoundation)) < 10 {
		t.Fatalf("expected foundation packages, got %d", len(catalogByLayer(kernel.LayerFoundation)))
	}
	if len(catalogByLayer(kernel.LayerIntelligence)) != 3 {
		t.Fatalf("expected 3 intelligence packages, got %d", len(catalogByLayer(kernel.LayerIntelligence)))
	}
	if len(catalogByLayer(kernel.LayerAddon)) < 20 {
		t.Fatalf("expected addon packages, got %d", len(catalogByLayer(kernel.LayerAddon)))
	}

	ai, ok := catalogLookup("ai")
	if !ok || ai.Layer != kernel.LayerIntelligence || ai.EffectiveKind() != kernel.KindService {
		t.Fatal("ai should be an intelligence service")
	}
	rag, ok := catalogLookup("rag")
	if !ok || rag.Layer != kernel.LayerIntelligence || rag.EffectiveKind() != kernel.KindLibrary {
		t.Fatal("rag should be an intelligence library")
	}
	agent, ok := catalogLookup("agent")
	if !ok || agent.Layer != kernel.LayerIntelligence || agent.EffectiveKind() != kernel.KindLibrary {
		t.Fatal("agent should be an intelligence library")
	}
	if _, ok := catalogLookup("container"); !ok {
		t.Fatal("primitives must still resolve through catalogLookup")
	}
	if _, ok := kernel.LookupPackage("auth"); ok {
		t.Fatal("kernel must not know auth")
	}
}

func TestEcosystemCatalogAddonKinds(t *testing.T) {
	services := 0
	libraries := 0
	for _, p := range catalogByLayer(kernel.LayerAddon) {
		switch p.EffectiveKind() {
		case kernel.KindService:
			services++
		case kernel.KindLibrary:
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
	mongo, ok := catalogLookup("mongo")
	if !ok || mongo.EffectiveKind() != kernel.KindService {
		t.Fatal("mongo should be a service addon")
	}
	collection, ok := catalogLookup("collection")
	if !ok || collection.EffectiveKind() != kernel.KindLibrary {
		t.Fatal("collection should be a library addon")
	}
	oauth, ok := catalogLookup("oauth")
	if !ok || oauth.EffectiveKind() != kernel.KindService {
		t.Fatal("oauth should be a service addon")
	}
	libs := catalogLibraries()
	if len(libs) < 10 {
		t.Fatalf("expected library catalog, got %d", len(libs))
	}
}
