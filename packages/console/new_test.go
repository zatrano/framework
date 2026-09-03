package console

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseNewArgs(t *testing.T) {
	dir, mod, replace, err := parseNewArgs([]string{"demo", "--module", "example.com/demo"})
	if err != nil {
		t.Fatal(err)
	}
	if dir != "demo" || mod != "example.com/demo" || replace != "" {
		t.Fatalf("dir=%q mod=%q replace=%q", dir, mod, replace)
	}
	if _, _, _, err := parseNewArgs(nil); err == nil {
		t.Fatal("expected usage error")
	}
}

func TestSanitizeModule(t *testing.T) {
	if got := sanitizeModule("My App"); got != "myapp" {
		t.Fatalf("got %q", got)
	}
	if got := sanitizeModule("github.com/acme/shop"); got != "github.com/acme/shop" {
		t.Fatalf("got %q", got)
	}
}

func TestNewScaffoldsBuildableApp(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "demo")
	cmd := &NewCommand{}
	if err := cmd.Handle([]string{dest, "--module", "example.com/demo", "--replace", root}); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dest, "cmd", "app", "main.go")
	body, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "__MODULE__") || strings.Contains(text, "__APP_NAME__") {
		t.Fatalf("placeholders left in main.go:\n%s", text)
	}
	if !strings.Contains(text, "example.com/demo/app/providers") {
		t.Fatalf("expected module import in main.go:\n%s", text)
	}
	modBytes, err := os.ReadFile(filepath.Join(dest, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	modText := string(modBytes)
	if !strings.Contains(modText, "module example.com/demo") {
		t.Fatalf("go.mod:\n%s", modText)
	}
	if strings.Contains(modText, "__FRAMEWORK_VERSION__") || strings.Contains(modText, "__REPLACE_LINE__") {
		t.Fatalf("placeholders left in go.mod:\n%s", modText)
	}
	agents, err := os.ReadFile(filepath.Join(dest, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "Router") || !strings.Contains(string(agents), "zatrano doctor") {
		t.Fatalf("AGENTS.md missing describe-derived content:\n%s", agents)
	}
	build := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "app.exe"), "./cmd/app")
	build.Dir = dest
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
}

func TestRenameTemplatePath(t *testing.T) {
	if got := renameTemplatePath("go.mod.tmpl"); got != "go.mod" {
		t.Fatalf("got %q", got)
	}
	if got := renameTemplatePath("cmd/app/main.go.tmpl"); got != "cmd/app/main.go" {
		t.Fatalf("got %q", got)
	}
}
