package addons_test

import (
	"testing"

	_ "github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
)

func TestRegistryEntriesAreCatalogServices(t *testing.T) {
	catalogServices := map[string]kernel.PackageInfo{}
	for _, layer := range []kernel.Layer{kernel.LayerAddon, kernel.LayerIntelligence} {
		for _, p := range kernel.PackagesByLayer(layer) {
			if p.EffectiveKind() != kernel.KindService {
				continue
			}
			catalogServices[p.Name] = p
		}
	}

	if len(addons.Available()) == 0 {
		t.Fatal("expected at least in-tree ai in registry")
	}
	for _, m := range addons.Available() {
		info, ok := catalogServices[m.Name]
		if !ok {
			t.Errorf("registry package %q missing from catalog as KindService", m.Name)
			continue
		}
		if info.Heavy != m.Heavy {
			t.Errorf("package %q heavy mismatch: catalog=%v registry=%v", m.Name, info.Heavy, m.Heavy)
		}
	}
}

func TestLibraryAddonsAreNotInRegistry(t *testing.T) {
	for _, layer := range []kernel.Layer{kernel.LayerAddon, kernel.LayerIntelligence} {
		for _, p := range kernel.PackagesByLayer(layer) {
			if p.EffectiveKind() != kernel.KindLibrary {
				continue
			}
			if _, ok := addons.Lookup(p.Name); ok {
				t.Errorf("library package %q should not be in provider registry", p.Name)
			}
		}
	}
}

func TestInTreeAIIsRegistered(t *testing.T) {
	if _, ok := addons.Lookup("ai"); !ok {
		t.Fatal("expected ai in registry")
	}
}
