package addons_test

import (
	"testing"

	_ "github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
)

func TestRegistryMatchesCatalogServiceAddons(t *testing.T) {
	catalogServices := map[string]kernel.PackageInfo{}
	for _, layer := range []kernel.Layer{kernel.LayerAddon, kernel.LayerIntelligence} {
		for _, p := range kernel.PackagesByLayer(layer) {
			if p.EffectiveKind() != kernel.KindService {
				continue
			}
			catalogServices[p.Name] = p
		}
	}

	for _, m := range addons.Available() {
		info, ok := catalogServices[m.Name]
		if !ok {
			t.Errorf("registry package %q missing from catalog as KindService addon", m.Name)
			continue
		}
		if info.Heavy != m.Heavy {
			t.Errorf("package %q heavy mismatch: catalog=%v registry=%v", m.Name, info.Heavy, m.Heavy)
		}
		delete(catalogServices, m.Name)
	}
	for name := range catalogServices {
		t.Errorf("catalog KindService addon %q missing from bootstrap/addons registry", name)
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

func TestEnabledAddonsAreKnown(t *testing.T) {
	for _, name := range []string{"mongo", "oauth", "webauthn", "ai", "octane"} {
		if _, ok := addons.Lookup(name); !ok {
			t.Errorf("expected %q in registry", name)
		}
	}
}
