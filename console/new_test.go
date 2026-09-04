package console

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseNewArgs(t *testing.T) {
	dir, mod, replace, minimal, err := parseNewArgs([]string{"demo", "--module", "example.com/demo"})
	if err != nil {
		t.Fatal(err)
	}
	if dir != "demo" || mod != "example.com/demo" || replace != "" || minimal {
		t.Fatalf("dir=%q mod=%q replace=%q minimal=%v", dir, mod, replace, minimal)
	}
	_, _, _, min, err := parseNewArgs([]string{"demo", "--minimal"})
	if err != nil {
		t.Fatal(err)
	}
	if !min {
		t.Fatal("expected --minimal")
	}
	if _, _, _, _, err := parseNewArgs(nil); err == nil {
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
	root, err := filepath.Abs("..")
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
	if _, err := os.Stat(filepath.Join(dest, "app", "views", "welcome.html")); err != nil {
		t.Fatalf("expected app/views/welcome.html: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "app", "database", "migrations", "migrations.go")); err != nil {
		t.Fatalf("expected app/database/migrations: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "app", "localization", "en")); err != nil {
		t.Fatalf("expected app/localization: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "bootstrap", "addons.go")); err != nil {
		t.Fatalf("expected bootstrap/addons.go: %v", err)
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
	if !strings.Contains(modText, "replace github.com/zatrano/framework =>") {
		t.Fatalf("missing framework replace:\n%s", modText)
	}
	if !strings.Contains(modText, "replace github.com/zatrano/packages =>") {
		t.Fatalf("missing packages replace:\n%s", modText)
	}
	if !strings.Contains(modText, "replace github.com/zatrano/packages/database/driver/sqlite =>") {
		t.Fatalf("missing sqlite driver replace:\n%s", modText)
	}
	envEx, err := os.ReadFile(filepath.Join(dest, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(envEx), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "DB_CONNECTION=") && !strings.HasPrefix(trim, "#") {
			t.Fatalf("new apps must not set DB_CONNECTION by default:\n%s", envEx)
		}
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

func TestNewMinimalHasNoPackageDeps(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "lite")
	cmd := &NewCommand{}
	if err := cmd.Handle([]string{dest, "--module", "example.com/lite", "--replace", root, "--minimal"}); err != nil {
		t.Fatal(err)
	}
	walk := exec.Command("go", "list", "-deps", "./cmd/app")
	walk.Dir = dest
	out, err := walk.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps: %v\n%s", err, out)
	}
	text := string(out)
	for _, pkg := range []string{
		"github.com/zatrano/packages/session",
		"github.com/zatrano/packages/database",
		"github.com/zatrano/packages/view",
	} {
		if strings.Contains(text, pkg) {
			t.Fatalf("minimal app must not depend on %s\n%s", pkg, text)
		}
	}
	if strings.Contains(text, "github.com/zatrano/packages/") {
		t.Fatalf("minimal app has packages deps:\n%s", text)
	}
	build := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "lite.exe"), "./cmd/app")
	build.Dir = dest
	bout, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, bout)
	}
}

func TestFrameworkGoModVersion(t *testing.T) {
	if got := frameworkGoModVersion("2.0.0-dev"); got != "v0.0.0" {
		t.Fatalf("got %q", got)
	}
	if got := frameworkGoModVersion("1.6.6"); got != "v1.6.6" {
		t.Fatalf("got %q", got)
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
