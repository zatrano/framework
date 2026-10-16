package kernel_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel"
)

func TestProductAndModuleIdentity(t *testing.T) {
	root := moduleRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	version := strings.TrimSpace(string(raw))
	if version != "3.2.0" {
		t.Fatalf("VERSION=%q want 3.2.0", version)
	}

	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	path := ""
	for _, line := range strings.Split(string(mod), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			path = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			break
		}
	}
	if path != "github.com/zatrano/framework/v3" {
		t.Fatalf("module path=%q", path)
	}

	app := kernel.NewApplication(root)
	if got := app.Version(); got != version {
		t.Fatalf("Version()=%q want %q", got, version)
	}
}

// Architecture: kernel must not import github.com/zatrano/packages.

func TestFrameworkDoesNotImportPackagesModule(t *testing.T) {
	root := moduleRoot(t)
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := filepath.Base(path)
			if name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if imp == "github.com/zatrano/packages" || strings.HasPrefix(imp, "github.com/zatrano/packages/") {
				t.Errorf("%s imports %s", rel, imp)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestKernelHasOnlyAllowedThirdPartyDependencies(t *testing.T) {
	root := moduleRoot(t)
	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	allowedMod := func(line string) bool {
		return strings.Contains(line, "github.com/zatrano/rawhttp") ||
			strings.Contains(line, "github.com/zatrano/canvas")
	}
	for _, line := range strings.Split(string(mod), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "require ") && line != "require (" {
			if allowedMod(line) {
				continue
			}
			t.Errorf("go.mod has a third-party require: %s", line)
		}
	}

	allowedImport := func(imp string) bool {
		return imp == "github.com/zatrano/rawhttp" || strings.HasPrefix(imp, "github.com/zatrano/rawhttp/") ||
			imp == "github.com/zatrano/canvas" || strings.HasPrefix(imp, "github.com/zatrano/canvas/")
	}

	fset := token.NewFileSet()
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := filepath.Base(path)
			// bench/ is its own module for rival measurements. Third-party
			// imports there stay out of the kernel module.
			if name == "vendor" || name == ".git" || name == "bench" {
				rel, relErr := filepath.Rel(root, path)
				if relErr == nil && (rel == "vendor" || rel == ".git" || rel == "bench") {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if !strings.Contains(imp, ".") {
				continue
			}
			if imp == "github.com/zatrano/framework/v3" || strings.HasPrefix(imp, "github.com/zatrano/framework/v3/") {
				continue
			}
			if allowedImport(imp) {
				continue
			}
			t.Errorf("%s imports third-party %s", rel, imp)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestKernelCatalogIsPrimitiveOnly(t *testing.T) {
	for _, p := range kernel.Catalog {
		if p.Layer != kernel.LayerPrimitive {
			t.Errorf("kernel.Catalog %q layer=%s want primitive", p.Name, p.Layer)
		}
	}
	for _, name := range []string{"auth", "database", "ai", "agent", "oauth"} {
		if _, ok := kernel.LookupPackage(name); ok {
			t.Errorf("kernel catalog must not contain %s", name)
		}
	}
}

func TestRemovedNegotiatePackageHasNoImportPath(t *testing.T) {
	root := moduleRoot(t)
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := filepath.Base(path)
			if name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if imp == "github.com/zatrano/packages/negotiate" || strings.HasPrefix(imp, "github.com/zatrano/packages/negotiate/") {
				t.Errorf("%s imports removed package %s", rel, imp)
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
