package console

import (
	"testing"

	"github.com/zatrano/framework/v2/manifest"
	"github.com/zatrano/framework/v2/registry"
)

func TestEcosystemCatalogBuildsRegistryIndex(t *testing.T) {
	hints := loadRegisterHints(t)
	docs := make([]manifest.Document, 0, len(ecosystemCatalog))
	for _, p := range ecosystemCatalog {
		in := manifest.Input{
			Name:        p.Name,
			Kind:        string(p.EffectiveKind()),
			Layer:       string(p.Layer),
			Description: p.Description,
			Heavy:       p.Heavy,
		}
		if h, ok := hints[p.Name]; ok {
			in.Factory = h.factory
			in.CLI = h.cli
			in.Requires = h.requires
			in.Key = h.key
		}
		docs = append(docs, manifest.Derive(in))
	}
	idx, err := registry.FromDocuments(docs)
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Packages) != len(ecosystemCatalog) {
		t.Fatalf("index=%d catalog=%d", len(idx.Packages), len(ecosystemCatalog))
	}
	got, err := idx.Resolve(registry.Query{Name: "session", Version: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Package.Module != manifest.DefaultModule {
		t.Fatalf("session module=%q", got.Package.Module)
	}
	libs := idx.Search(registry.Filter{Kind: manifest.KindLibrary})
	if len(libs) < 30 {
		t.Fatalf("expected many libraries, got %d", len(libs))
	}
	heavy := true
	h := idx.Search(registry.Filter{Heavy: &heavy})
	if len(h) != 3 {
		t.Fatalf("heavy want 3 (mongo, webauthn, qr), got %d", len(h))
	}
}
