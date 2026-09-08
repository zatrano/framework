package tests

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhase11DoesNotIntroduceForbiddenArchitecture(t *testing.T) {
	root := moduleRoot(t)
	acquireDir := filepath.Join(root, "distribution", "acquire")
	err := filepath.WalkDir(acquireDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return walkErr
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(body)
		for _, ban := range []string{
			"func Apply(",
			"zatrano.lock",
			"go mod tidy",
			"type LifecycleManager",
			"type ZatranoContext",
		} {
			if strings.Contains(src, ban) {
				t.Errorf("%s contains %s", filepath.Base(path), ban)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	appSrc, err := os.ReadFile(filepath.Join(root, "bootstrap", "app.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(appSrc), "MeetsFrameworkMin") {
		t.Fatal("App() must not validate framework_min")
	}

	kernelSrc, err := os.ReadFile(filepath.Join(root, "kernel", "application.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(kernelSrc), "MeetsFrameworkMin") {
		t.Fatal("Bootstrap/Start must not validate framework_min")
	}
	if strings.Contains(string(kernelSrc), "github.com/zatrano/packages") {
		t.Fatal("framework must not import official packages")
	}

	registrySrc, err := os.ReadFile(filepath.Join(root, "bootstrap", "addons", "registry.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(registrySrc), "process-global") {
		t.Fatal("process-global addon registry must remain the documented model")
	}
}
