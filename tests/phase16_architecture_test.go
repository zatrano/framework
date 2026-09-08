package tests

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhase16ReferenceApplicationExists(t *testing.T) {
	root := moduleRoot(t)
	readme := filepath.Join(root, "examples", "reference", "README.md")
	if _, err := os.Stat(readme); err != nil {
		t.Fatalf("reference README missing: %v", err)
	}
	main := filepath.Join(root, "examples", "reference", "cmd", "reference", "main.go")
	if _, err := os.Stat(main); err != nil {
		t.Fatalf("reference main missing: %v", err)
	}
}

func TestPhase16ReferenceDoesNotImportPackagesModule(t *testing.T) {
	root := filepath.Join(moduleRoot(t), "examples", "reference")
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(moduleRoot(t), path)
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, `"`)
			if imp == "github.com/zatrano/packages" || strings.HasPrefix(imp, "github.com/zatrano/packages/") {
				t.Errorf("%s imports %s", filepath.ToSlash(rel), imp)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPhase16DoesNotIntroduceForbiddenArchitecture(t *testing.T) {
	root := moduleRoot(t)
	bans := []string{
		"type PackageManager",
		"type LifecycleManager",
		"type ApplicationManager",
		"type ConfigManager",
		"type RuntimeManager",
		"type ServiceLocator",
		"type DependencyContainer",
		"type WorkerManager",
		"func Apply(",
		"func Rollback(",
		"zatrano.lock",
	}
	err := filepath.WalkDir(filepath.Join(root, "examples", "reference"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(body)
		rel, _ := filepath.Rel(root, path)
		for _, ban := range bans {
			if strings.Contains(src, ban) {
				t.Errorf("%s contains %s", filepath.ToSlash(rel), ban)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mod), "github.com/zatrano/packages") {
		t.Fatal("framework must not require github.com/zatrano/packages")
	}

	kernelSrc, err := os.ReadFile(filepath.Join(root, "kernel", "application.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(kernelSrc), "MeetsFrameworkMin") {
		t.Fatal("Bootstrap/Start must not enforce framework_min")
	}

	ver, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(ver)) != "2.0.28" {
		t.Fatalf("VERSION=%q want 2.0.28", strings.TrimSpace(string(ver)))
	}
}
