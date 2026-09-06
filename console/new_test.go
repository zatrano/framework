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
	assertGeneratedConsoleRegisterABI(t, dest)
	enabledBody, err := os.ReadFile(filepath.Join(dest, "bootstrap", "enabled.go"))
	if err != nil {
		t.Fatalf("expected bootstrap/enabled.go: %v", err)
	}
	enabledText := string(enabledBody)
	for _, want := range []string{"RegisterEnablement", `"assets"`, `"health"`, `"localization"`, `"view"`} {
		if !strings.Contains(enabledText, want) {
			t.Fatalf("enabled.go missing %q:\n%s", want, enabledText)
		}
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
	assertGeneratedFrameworkRequire(t, modText)
	if !strings.Contains(modText, "replace github.com/zatrano/framework/v2 =>") {
		t.Fatalf("missing framework replace:\n%s", modText)
	}
	if !strings.Contains(modText, "replace github.com/zatrano/packages =>") {
		t.Fatalf("missing packages replace:\n%s", modText)
	}
	for _, rel := range []string{
		"database/driver/sqlite",
		"database/driver/mysql",
		"database/driver/pgsql",
	} {
		want := "replace github.com/zatrano/packages/" + rel + " =>"
		if !strings.Contains(modText, want) {
			t.Fatalf("missing %s:\n%s", want, modText)
		}
	}
	envEx, err := os.ReadFile(filepath.Join(dest, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	envText := string(envEx)
	for _, line := range strings.Split(envText, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "DB_CONNECTION=") && !strings.HasPrefix(trim, "#") {
			t.Fatalf("new apps must not set DB_CONNECTION by default:\n%s", envEx)
		}
	}
	for _, key := range []string{
		"SESSION_DRIVER=", "CACHE_STORE=", "QUEUE_CONNECTION=", "REDIS_HOST=",
		"MAIL_MAILER=", "STRIPE_", "AI_DRIVER=", "MONGO_URI=", "OAUTH_",
		"GOOGLE_CLIENT_", "WEBAUTHN_", "LOG_CHANNEL=",
	} {
		if strings.Contains(envText, key) {
			t.Fatalf("generated .env.example must be kernel-only, found %q:\n%s", key, envEx)
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
	testCmd := exec.Command("go", "test", "./tests")
	testCmd.Dir = dest
	testOut, err := testCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./tests: %v\n%s", err, testOut)
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
	modBytes, err := os.ReadFile(filepath.Join(dest, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	assertGeneratedFrameworkRequire(t, string(modBytes))
	assertGeneratedConsoleRegisterABI(t, dest)
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
	enabledBody, err := os.ReadFile(filepath.Join(dest, "bootstrap", "enabled.go"))
	if err != nil {
		t.Fatalf("expected bootstrap/enabled.go: %v", err)
	}
	enabledText := string(enabledBody)
	if !strings.Contains(enabledText, "RegisterEnablement") {
		t.Fatalf("minimal enabled.go must register the manifest:\n%s", enabledText)
	}
	if strings.Contains(enabledText, `"health"`) || strings.Contains(enabledText, `"view"`) {
		t.Fatalf("minimal enabled.go must be empty:\n%s", enabledText)
	}
	build := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "lite.exe"), "./cmd/app")
	build.Dir = dest
	bout, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, bout)
	}
}

func TestFrameworkGoModVersion(t *testing.T) {
	if got := frameworkGoModVersion("2.0.0"); got != "v2.0.0" {
		t.Fatalf("2.0.0: got %q", got)
	}
	if got := frameworkGoModVersion("2.0.0-dev"); got != "v2.0.0" {
		t.Fatalf("2.0.0-dev: got %q", got)
	}
	if got := frameworkGoModVersion("1.6.6"); got != "v1.6.6" {
		t.Fatalf("1.6.6: got %q", got)
	}
}

func assertGeneratedFrameworkRequire(t *testing.T, modText string) {
	t.Helper()
	if !strings.Contains(modText, "github.com/zatrano/framework/v2 v2.0.0") {
		t.Fatalf("generated require must be v2.0.0:\n%s", modText)
	}
}

func assertGeneratedConsoleRegisterABI(t *testing.T, dest string) {
	t.Helper()
	kernelPath := filepath.Join(dest, "app", "console", "kernel.go")
	body, err := os.ReadFile(kernelPath)
	if err != nil {
		t.Fatalf("expected %s: %v", kernelPath, err)
	}
	text := string(body)
	if !strings.Contains(text, "func Register(cli *coreconsole.Application, app contracts.App)") {
		t.Fatalf("generated Register must take contracts.App:\n%s", text)
	}
	if strings.Contains(text, "*kernel.Application") {
		t.Fatalf("generated Register must not leak *kernel.Application:\n%s", text)
	}
	if strings.Contains(text, `"github.com/zatrano/framework/v2/kernel"`) {
		t.Fatalf("generated kernel.go must not import kernel:\n%s", text)
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
