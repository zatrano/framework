package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDeriveSessionService(t *testing.T) {
	d := Derive(Input{
		Name:        "session",
		Kind:        KindService,
		Layer:       LayerFoundation,
		Description: "HTTP sessions",
		Factory:     true,
	})
	if d.Import != DefaultModule+"/session" {
		t.Fatalf("import=%q", d.Import)
	}
	if d.Module != DefaultModule {
		t.Fatalf("module=%q", d.Module)
	}
	if d.Provider != ProviderFactory {
		t.Fatalf("provider=%q", d.Provider)
	}
	if err := Validate(d); err != nil {
		t.Fatal(err)
	}
}

func TestDeriveHeavyMongo(t *testing.T) {
	d := Derive(Input{
		Name:        "mongo",
		Kind:        KindService,
		Layer:       LayerAddon,
		Description: "MongoDB client",
		Heavy:       true,
		Factory:     true,
	})
	if d.Module != DefaultModule+"/mongo" || d.Import != d.Module {
		t.Fatalf("module=%q import=%q", d.Module, d.Import)
	}
	if err := Validate(d); err != nil {
		t.Fatal(err)
	}
}

func TestDeriveLibraryCLI(t *testing.T) {
	d := Derive(Input{
		Name:        "factory",
		Kind:        KindLibrary,
		Layer:       LayerAddon,
		Description: "Model factories",
		CLI:         true,
	})
	if d.Provider != ProviderCLI {
		t.Fatalf("provider=%q", d.Provider)
	}
	for _, cap := range d.Capabilities {
		if cap == CapEnable {
			t.Fatal("library must not be enableable")
		}
	}
	if err := Validate(d); err != nil {
		t.Fatal(err)
	}
}

func TestDeriveConsoleStaysOnFrameworkModule(t *testing.T) {
	d := Derive(Input{
		Name:        "console",
		Kind:        KindService,
		Layer:       LayerFoundation,
		Description: "CLI application",
	})
	if d.Import != FrameworkConsole || d.Module != FrameworkModule {
		t.Fatalf("console import=%q module=%q", d.Import, d.Module)
	}
	if err := Validate(d); err != nil {
		t.Fatal(err)
	}
}

func TestParseRoundTripTestdata(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) < 5 {
		t.Fatalf("expected fixture files, got %d", len(matches))
	}
	for _, path := range matches {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		d, err := Parse(raw)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		enc, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Parse(enc); err != nil {
			t.Fatalf("%s reparse: %v", path, err)
		}
	}
}

func TestParseIgnoresUnknownFields(t *testing.T) {
	raw := []byte(`{
		"schema": "zatrano.package/v1",
		"name": "collection",
		"import": "github.com/zatrano/packages/collection",
		"kind": "library",
		"layer": "addon",
		"description": "Collection helpers",
		"publisher": "deferred",
		"license": "MIT"
	}`)
	if _, err := Parse(raw); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsLibraryEnable(t *testing.T) {
	d := Derive(Input{Name: "collection", Kind: KindLibrary, Layer: LayerAddon, Description: "x"})
	d.Capabilities = append(d.Capabilities, CapEnable)
	if err := Validate(d); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateRejectsUnknownSchema(t *testing.T) {
	d := Derive(Input{Name: "session", Kind: KindService, Layer: LayerFoundation, Description: "x"})
	d.Schema = "zatrano.package/v0"
	if err := Validate(d); err == nil {
		t.Fatal("expected error")
	}
}

func TestFrameworkMinOptional(t *testing.T) {
	d := Derive(Input{Name: "session", Kind: KindService, Layer: LayerFoundation, Description: "x", FrameworkMin: "2.0.1"})
	if err := Validate(d); err != nil {
		t.Fatal(err)
	}
	d.FrameworkMin = "not-a-version"
	if err := Validate(d); err == nil {
		t.Fatal("expected invalid framework_min")
	}
}
