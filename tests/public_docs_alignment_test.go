package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicDocsDescribeReleasedEcosystem(t *testing.T) {
	root := moduleRoot(t)
	files := []string{
		"README.md",
		"PACKAGES.md",
		"distribution/registry/SPEC.md",
		"distribution/acquire/SPEC.md",
	}
	bans := []string{
		"github.com/zatrano/packages@main",
		"github.com/zatrano/framework/v2@latest",
		"uses channel `main` until tagged",
		"the module is not tagged `v2.x`",
		"Today that stream is channel `main`",
	}
	for _, rel := range files {
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, ban := range bans {
			if strings.Contains(text, ban) {
				t.Errorf("%s still describes the pre-release model: %q", rel, ban)
			}
		}
		if !strings.Contains(text, "github.com/zatrano/packages@v1.7.1") && !strings.Contains(text, "v1.7.1") {
			t.Errorf("%s must name current packages v1.7.1", rel)
		}
	}

	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(readme)
	for _, want := range []string{
		"github.com/zatrano/framework/v2@v2.1.0",
		"github.com/zatrano/packages@v1.7.1",
		"Enabled ∩ Imported",
		"Expand Requires",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("README.md missing %q", want)
		}
	}

	catalog, err := os.ReadFile(filepath.Join(root, "console", "catalog.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(catalog)
	if !strings.Contains(src, `{Name: "redisx", Layer: kernel.LayerFoundation, Kind: kernel.KindLibrary`) {
		t.Fatal("console catalog must list redisx as KindLibrary")
	}
	if strings.Contains(src, `Name: "redisx", Layer: kernel.LayerFoundation, Kind: kernel.KindService`) {
		t.Fatal("console catalog must not list redisx as KindService")
	}
}
