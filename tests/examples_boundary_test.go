package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrameworkDoesNotVendorExampleApplications(t *testing.T) {
	root := moduleRoot(t)
	dir := filepath.Join(root, "examples")
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		t.Fatal("example applications belong in github.com/zatrano/examples, not this module")
	}
}

func TestExamplesBoundaryDoesNotIntroduceForbiddenArchitecture(t *testing.T) {
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
	files := []string{
		filepath.Join("console", "console.go"),
		filepath.Join("console", "new.go"),
		filepath.Join("kernel", "application.go"),
		filepath.Join("kernel", "env", "env.go"),
		filepath.Join("contracts", "app.go"),
		filepath.Join("bootstrap", "app.go"),
	}
	for _, rel := range files {
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		src := string(body)
		for _, ban := range bans {
			if strings.Contains(src, ban) {
				t.Errorf("%s contains %s", filepath.ToSlash(rel), ban)
			}
		}
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
	if strings.TrimSpace(string(ver)) != "2.1.0" {
		t.Fatalf("VERSION=%q want 2.1.0", strings.TrimSpace(string(ver)))
	}
}
