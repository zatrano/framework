package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	if version != "3.2.4" {
		t.Fatalf("VERSION=%q want 3.2.4", version)
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
	order := []string{
		"## 3.2.4 -", "## 3.2.3 -", "## 3.2.2 -", "## 3.2.1 -", "## 3.2.0 -",
		"## 3.1.7 -", "## 3.1.6 -", "## 3.1.5 -", "## 3.1.4 -", "## 3.1.3 -", "## 3.1.2 -", "## 3.1.1 -", "## 3.1.0 -",
		"## 3.0.1 -", "## 3.0.0 -", "## 3.0.0-rc.1", "## 2.8.1", "## 2.8.0",
	}
	last := -1
	for _, heading := range order {
		at := strings.Index(text, heading)
		if at < 0 {
			t.Fatalf("CHANGELOG is missing %q", heading)
		}
		if at <= last {
			t.Fatalf("CHANGELOG must list %q after the newer releases: %v", heading, order)
		}
		last = at
	}

	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "version-3.2.4-green") {
		t.Fatal("README badge must show 3.2.4")
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
	if !strings.Contains(string(mod), "github.com/zatrano/framework/v3 v3.2.4") {
		t.Fatalf("generated go.mod must require v3.2.4:\n%s", mod)
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
		Require []struct {
			Path    string `json:"Path"`
			Version string `json:"Version"`
		} `json:"Require"`
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
	var canvas string
	for _, r := range doc.Require {
		if r.Path == "github.com/zatrano/canvas" {
			canvas = r.Version
		}
	}
	if canvas != "v1.0.1" {
		t.Fatalf("go.mod must require canvas v1.0.1, got %s", canvas)
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
