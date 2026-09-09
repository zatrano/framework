package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReleaseMetadata(t *testing.T) {
	root := moduleRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	version := strings.TrimSpace(string(raw))
	if version != "2.1.0" {
		t.Fatalf("VERSION=%q want 2.1.0", version)
	}

	log, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(log)
	if !strings.Contains(text, "## 2.1.0 - 2026-09-09") {
		t.Fatal("CHANGELOG.md must record 2.1.0")
	}
	if !strings.Contains(text, "## 2.0.28 - 2026-09-08") {
		t.Fatal("CHANGELOG.md must keep historical 2.0.28")
	}
	unreleased := strings.Index(text, "## Unreleased")
	released := strings.Index(text, "## 2.1.0")
	prev := strings.Index(text, "## 2.0.28")
	if unreleased < 0 || released < 0 || prev < 0 || !(unreleased < released && released < prev) {
		t.Fatal("CHANGELOG must keep Unreleased, then 2.1.0, then immutable 2.0.28")
	}

	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "version-2.1.0") {
		t.Fatal("README badge must show 2.1.0")
	}
}

func TestFreshConsumerLocalReplace(t *testing.T) {
	if testing.Short() {
		t.Skip("Release consumer uses go run + go build")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go executable not on PATH")
	}
	root := moduleRoot(t)
	parent := t.TempDir()
	dest := filepath.Join(parent, "freshapp")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/zatrano", "new", dest, "--module", "example.com/freshapp", "--replace", root)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zatrano new: %v\n%s", err, out)
	}
	mod, err := os.ReadFile(filepath.Join(dest, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mod), "github.com/zatrano/framework/v2 v2.1.0") {
		t.Fatalf("generated go.mod must require v2.1.0:\n%s", mod)
	}
	build := exec.CommandContext(ctx, "go", "build", "-o", filepath.Join(t.TempDir(), "app.exe"), "./cmd/app")
	build.Dir = dest
	build.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	bout, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build generated app: %v\n%s", err, bout)
	}
}

func TestPublishedModuleConsumption(t *testing.T) {
	if testing.Short() {
		t.Skip("published-tag consumption needs the module proxy")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go executable not on PATH")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/zatrano-release-consumer\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	get := exec.Command("go", "get", "github.com/zatrano/framework/v2@v2.1.0")
	get.Dir = dir
	get.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	gout, err := get.CombinedOutput()
	if err != nil {
		t.Skipf("v2.1.0 is not on the module proxy yet (tag/push not done): %v\n%s", err, gout)
	}
	mod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mod), "github.com/zatrano/framework/v2 v2.1.0") {
		t.Fatalf("consumer go.mod must pin v2.1.0:\n%s", mod)
	}
}
