package addons

import (
	"testing"

	"github.com/zatrano/framework/core"
)

func TestRegistryMatchesCatalogServiceAddons(t *testing.T) {
	catalogServices := map[string]core.PackageInfo{}
	for _, p := range core.PackagesByLayer(core.LayerAddon) {
		if p.EffectiveKind() != core.KindService {
			continue
		}
		catalogServices[p.Name] = p
	}

	for _, m := range Available() {
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
	for _, p := range core.PackagesByLayer(core.LayerAddon) {
		if p.EffectiveKind() != core.KindLibrary {
			continue
		}
		if _, ok := Lookup(p.Name); ok {
			t.Errorf("library addon %q should not be in provider registry", p.Name)
		}
	}
}

func TestEnabledAddonsAreKnown(t *testing.T) {
	for _, name := range []string{"mongo", "oauth", "webauthn", "ai", "octane"} {
		if _, ok := Lookup(name); !ok {
			t.Errorf("expected %q in registry", name)
		}
	}
}
