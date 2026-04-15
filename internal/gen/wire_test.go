package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWirePatch_roundTrip(t *testing.T) {
	const initial = "package routes\n\nimport (\n\t\"github.com/gofiber/fiber/v3\"\n\n\t\"github.com/zatrano/framework/pkg/core\"\n\t// zatrano:wire:imports:start\n\t// zatrano:wire:imports:end\n)\n\nfunc Register(a *core.App, app *fiber.App) {\n\t// zatrano:wire:register:start\n\t// zatrano:wire:register:end\n}\n"
	tmp := t.TempDir()
	p := filepath.Join(tmp, "register.go")
	if err := os.WriteFile(p, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WirePatch(p, "example.com/app", "modules/foo_bar", "foo_bar", true, true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, `"example.com/app/modules/foo_bar"`) {
		t.Fatalf("missing import: %s", s)
	}
	if !strings.Contains(s, "foo_bar.Register(a, app)") || !strings.Contains(s, "foo_bar.RegisterCRUD(a, app)") {
		t.Fatalf("missing calls: %s", s)
	}
	if err := WirePatch(p, "example.com/app", "modules/foo_bar", "foo_bar", true, true); err != nil {
		t.Fatal(err)
	}
	b2, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b2) != s {
		t.Fatal("idempotent second patch changed file")
	}
}

func TestWireTargetsFromModuleDir(t *testing.T) {
	tmp := t.TempDir()
	mod := filepath.Join(tmp, "go.mod")
	if err := os.WriteFile(mod, []byte("module test.wire\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(tmp, "modules", "foo_bar")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "register.go"), []byte("package foo_bar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg, crud, dir, err := WireTargetsFromModuleDir(tmp, "modules", "foo-bar")
	if err != nil {
		t.Fatal(err)
	}
	if !reg || crud {
		t.Fatalf("want Register only, got reg=%v crud=%v", reg, crud)
	}
	if err := os.WriteFile(filepath.Join(base, "crud_register.go"), []byte("package foo_bar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg, crud, _, err = WireTargetsFromModuleDir(tmp, "modules", "foo_bar")
	if err != nil {
		t.Fatal(err)
	}
	if !reg || !crud {
		t.Fatalf("want both, got reg=%v crud=%v", reg, crud)
	}
	_ = dir
}

func TestResolveWireFile_prefersInternalRoutes(t *testing.T) {
	tmp := t.TempDir()
	internalDir := filepath.Join(tmp, "internal", "routes")
	if err := os.MkdirAll(internalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(internalDir, "register.go")
	fw := filepath.Join(tmp, "pkg", "server", "register_modules.go")
	content := []byte("package x\n\nimport (\n\t\"y\"\n\t// zatrano:wire:imports:start\n\t// zatrano:wire:imports:end\n)\n\nfunc Register() {\n\t// zatrano:wire:register:start\n\t// zatrano:wire:register:end\n}\n")
	if err := os.WriteFile(app, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(fw), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fw, content, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveWireFile(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if got != app {
		t.Fatalf("expected internal/routes first, got %s", got)
	}
}
