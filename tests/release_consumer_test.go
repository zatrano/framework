package tests

import (
	"context"
	"encoding/json"
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
	if version != "3.1.0" {
		t.Fatalf("VERSION=%q want 3.1.0", version)
	}

	log, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(log)
	if !strings.Contains(text, "## 2.8.1 - 2026-09-20") {
		t.Fatal("CHANGELOG.md must record 2.8.1")
	}
	if !strings.Contains(text, "## 2.8.0 - 2026-09-20") {
		t.Fatal("CHANGELOG.md must keep historical 2.8.0")
	}
	if !strings.Contains(text, "## 2.7.0 - 2026-09-19") {
		t.Fatal("CHANGELOG.md must keep historical 2.7.0")
	}
	if !strings.Contains(text, "## 2.6.4 - 2026-09-19") {
		t.Fatal("CHANGELOG.md must keep historical 2.6.4")
	}
	if !strings.Contains(text, "## 2.6.3 - 2026-09-19") {
		t.Fatal("CHANGELOG.md must keep historical 2.6.3")
	}
	if !strings.Contains(text, "## 2.6.2 - 2026-09-19") {
		t.Fatal("CHANGELOG.md must keep historical 2.6.2")
	}
	if !strings.Contains(text, "## 2.6.1 - 2026-09-19") {
		t.Fatal("CHANGELOG.md must keep historical 2.6.1")
	}
	if !strings.Contains(text, "## 2.6.0 - 2026-09-15") {
		t.Fatal("CHANGELOG.md must keep historical 2.6.0")
	}
	if !strings.Contains(text, "## 2.5.1 - 2026-09-15") {
		t.Fatal("CHANGELOG.md must keep historical 2.5.1")
	}
	if !strings.Contains(text, "## 2.5.0 - 2026-09-15") {
		t.Fatal("CHANGELOG.md must keep historical 2.5.0")
	}
	if !strings.Contains(text, "## 2.4.0 - 2026-09-15") {
		t.Fatal("CHANGELOG.md must keep historical 2.4.0")
	}
	if !strings.Contains(text, "## 2.3.1 - 2026-09-11") {
		t.Fatal("CHANGELOG.md must keep historical 2.3.1")
	}
	if !strings.Contains(text, "## 2.3.0 - 2026-09-11") {
		t.Fatal("CHANGELOG.md must keep historical 2.3.0")
	}
	if !strings.Contains(text, "## 2.2.1 - 2026-09-11") {
		t.Fatal("CHANGELOG.md must keep historical 2.2.1")
	}
	if !strings.Contains(text, "## 2.2.0 - 2026-09-10") {
		t.Fatal("CHANGELOG.md must record 2.2.0")
	}
	if !strings.Contains(text, "## 2.1.0 - 2026-09-09") {
		t.Fatal("CHANGELOG.md must keep historical 2.1.0")
	}
	if !strings.Contains(text, "## 2.0.28 - 2026-09-08") {
		t.Fatal("CHANGELOG.md must keep historical 2.0.28")
	}
	unreleased := strings.Index(text, "## Unreleased")
	v302 := strings.Index(text, "## 3.1.0 -")
	v301 := strings.Index(text, "## 3.0.1 -")
	stable := strings.Index(text, "## 3.0.0 -")
	rc := strings.Index(text, "## 3.0.0-rc.1")
	released := strings.Index(text, "## 2.8.1")
	prev := strings.Index(text, "## 2.8.0")
	if unreleased < 0 || v302 < 0 || v301 < 0 || stable < 0 || rc < 0 || released < 0 || prev < 0 || !(unreleased < v302 && v302 < v301 && v301 < stable && stable < rc && rc < released && released < prev) {
		t.Fatal("CHANGELOG must keep Unreleased, then 3.1.0, then 3.0.1, then 3.0.0, then 3.0.0-rc.1, then 2.8.1, then immutable 2.8.0")
	}

	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "version-3.1.0-green") {
		t.Fatal("README badge must show 3.1.0")
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
	if !strings.Contains(string(mod), "github.com/zatrano/framework/v3 v3.1.0") {
		t.Fatalf("generated go.mod must require v3.1.0:\n%s", mod)
	}
	build := exec.CommandContext(ctx, "go", "build", "-o", filepath.Join(t.TempDir(), "app.exe"), "./cmd/app")
	build.Dir = dest
	build.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	bout, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build generated app: %v\n%s", err, bout)
	}
}

func TestReleaseModuleHasNoReplace(t *testing.T) {
	root := moduleRoot(t)
	cmd := exec.Command("go", "mod", "edit", "-json")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Replace []*json.RawMessage `json:"Replace"`
		Retract []struct {
			Low string `json:"Low"`
		} `json:"Retract"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Replace) != 0 {
		t.Fatalf("go.mod Replace must be empty, got %s", out)
	}
	retracted := false
	for _, r := range doc.Retract {
		if r.Low == "v3.0.0" {
			retracted = true
		}
	}
	if !retracted {
		t.Fatalf("go.mod must retract v3.0.0:\n%s", out)
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
	get := exec.Command("go", "get", "github.com/zatrano/framework/v3@v3.0.1")
	get.Dir = dir
	get.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	gout, err := get.CombinedOutput()
	if err != nil {
		t.Skipf("v3.0.1 is not on the module proxy yet (tag/push not done): %v\n%s", err, gout)
	}
	mod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mod), "github.com/zatrano/framework/v3 v3.0.1") {
		t.Fatalf("consumer go.mod must pin v3.0.1:\n%s", mod)
	}
}
