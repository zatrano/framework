package describe

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/distribution/manifest"
)

var reAddonName = regexp.MustCompile(`addons\.Register\(\s*addons\.Meta\{[^}]*Name:\s*"([a-z0-9]+)"`)

func packagesRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	framework := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	for _, root := range []string{
		filepath.Join(framework, "packages"),
		filepath.Join(filepath.Dir(framework), "packages"),
	} {
		if st, err := os.Stat(root); err == nil && st.IsDir() {
			return root
		}
	}
	return ""
}

func TestRegisteredPackageNamesAreCatalogued(t *testing.T) {
	root := packagesRepoRoot(t)
	if root == "" {
		t.Skip("packages checkout not beside framework")
	}

	found := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		src := string(body)
		if !strings.Contains(src, "addons.Register") {
			return nil
		}
		for _, m := range reAddonName.FindAllStringSubmatch(src, -1) {
			if len(m) == 2 {
				found[m[1]] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) < 35 {
		t.Fatalf("expected many addons.Register names, got %d", len(found))
	}
	for name := range found {
		if _, ok := catalogLookup(name); !ok {
			t.Errorf("packages registers %q but console catalog does not list it", name)
		}
	}
}

func TestEcosystemCatalogHasPackageDirectories(t *testing.T) {
	root := packagesRepoRoot(t)
	if root == "" {
		t.Skip("packages checkout not beside framework")
	}
	for _, p := range ecosystemCatalog {
		if p.Name == "console" {
			continue // lives in the framework CLI, not packages/
		}
		rel := manifest.OfficialImportRel(p.Name)
		dir := filepath.Join(root, filepath.FromSlash(rel))
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			t.Errorf("catalog %q has no packages/%s directory", p.Name, rel)
		}
	}
	for _, internal := range []string{"bootutil"} {
		if _, ok := catalogLookup(internal); ok {
			t.Errorf("%q is internal and must not be in the consumer catalog", internal)
		}
	}
}
