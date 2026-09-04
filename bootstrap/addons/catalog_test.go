package addons_test

import (
	"testing"

	_ "github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
)

func TestRegistryEntriesAreCatalogServices(t *testing.T) {
	catalogServices := map[string]kernel.PackageInfo{}
	for _, layer := range []kernel.Layer{kernel.LayerAddon, kernel.LayerIntelligence, kernel.LayerFoundation} {
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
			continue
		}
		if info.Heavy != m.Heavy {
			t.Errorf("package %q heavy mismatch: catalog=%v registry=%v", m.Name, info.Heavy, m.Heavy)
		}
	}
}

func TestFrameworkBinaryRegistersNoPackages(t *testing.T) {
	if got := addons.Available(); len(got) != 0 {
		t.Fatalf("framework must not blank-import packages, got %v", addons.Names())
	}
}
