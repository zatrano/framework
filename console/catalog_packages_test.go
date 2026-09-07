package console

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var reAddonName = regexp.MustCompile(`addons\.Register\(\s*addons\.Meta\{[^}]*Name:\s*"([a-z0-9]+)"`)

func TestRegisteredPackageNamesAreCatalogued(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "packages")
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
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
	if len(found) < 40 {
		t.Fatalf("expected many addons.Register names, got %d", len(found))
	}
	for name := range found {
		if _, ok := catalogLookup(name); !ok {
			t.Errorf("packages registers %q but console catalog does not list it", name)
		}
	}
}

func TestEcosystemCatalogHasPackageDirectories(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "packages")
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		t.Skip("packages checkout not beside framework")
	}
	for _, p := range ecosystemCatalog {
		if p.Name == "console" {
			continue // lives in the framework CLI, not packages/
		}
		dir := filepath.Join(root, p.Name)
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			t.Errorf("catalog %q has no packages/%s directory", p.Name, p.Name)
		}
	}
	for _, internal := range []string{"bootutil"} {
		if _, ok := catalogLookup(internal); ok {
			t.Errorf("%q is internal and must not be in the consumer catalog", internal)
		}
	}
}
