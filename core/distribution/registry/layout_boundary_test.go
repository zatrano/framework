package registry_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDistributionProtocolLivesUnderDistribution(t *testing.T) {
	root := moduleRoot(t)
	for _, name := range []string{"acquire", "manifest", "registry"} {
		if st, err := os.Stat(filepath.Join(root, name)); err == nil && st.IsDir() {
			t.Errorf("%s/ must not sit at module root — use distribution/%s", name, name)
		}
		dir := filepath.Join(root, "core", "distribution", name)
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			t.Errorf("missing distribution/%s", name)
		}
	}
}

func TestRegistryDoesNotImportConsole(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "core", "distribution", "registry")
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if strings.Contains(imp, "/console") {
				t.Errorf("%s imports %s — CLI consumes registry, not the reverse", filepath.Base(path), imp)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("module root")
		}
		dir = parent
	}
}
