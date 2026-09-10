package console

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/console/generator"
)

func TestParseNewArgs(t *testing.T) {
	dir, mod, replace, scaffold, err := parseNewArgs([]string{"demo", "--module", "example.com/demo"})
	if err != nil {
		t.Fatal(err)
	}
	if dir != "demo" || mod != "example.com/demo" || replace != "" || scaffold != generator.ScaffoldEmpty {
		t.Fatalf("dir=%q mod=%q replace=%q scaffold=%q", dir, mod, replace, scaffold)
	}
	_, _, _, web, err := parseNewArgs([]string{"demo", "--web"})
	if err != nil {
		t.Fatal(err)
	}
	if web != generator.ScaffoldWeb {
		t.Fatalf("expected --web, got %q", web)
	}
	_, _, _, api, err := parseNewArgs([]string{"demo", "--api"})
	if err != nil {
		t.Fatal(err)
	}
	if api != generator.ScaffoldAPI {
		t.Fatalf("expected --api, got %q", api)
	}
	_, _, _, full, err := parseNewArgs([]string{"demo", "--full"})
	if err != nil {
		t.Fatal(err)
	}
	if full != generator.ScaffoldFull {
		t.Fatalf("expected --full, got %q", full)
	}
	if _, _, _, _, err := parseNewArgs(nil); err == nil {
		t.Fatal("expected usage error")
	}
}

func TestNewRejectsMinimal(t *testing.T) {
	_, _, _, _, err := parseNewArgs([]string{"demo", "--minimal"})
	if err == nil || !strings.Contains(err.Error(), "no longer a supported scaffold profile") {
		t.Fatalf("expected --minimal rejection, got %v", err)
	}
	for _, name := range []string{"empty", "--web", "--api", "--full"} {
		if !strings.Contains(newHelp, name) {
			t.Fatalf("new help must name %q:\n%s", name, newHelp)
		}
	}
}

func TestNewRejectsConflictingProfiles(t *testing.T) {
	for _, args := range [][]string{
		{"demo", "--web", "--api"},
		{"demo", "--web", "--full"},
		{"demo", "--api", "--full"},
	} {
		if _, _, _, _, err := parseNewArgs(args); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
			t.Fatalf("expected exclusive error for %v, got %v", args, err)
		}
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

func TestNewWebApplication(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "demo")
	cmd := &NewCommand{}
	if err := cmd.Handle([]string{dest, "--module", "example.com/demo", "--replace", root, "--web"}); err != nil {
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
	addonsBody, err := os.ReadFile(filepath.Join(dest, "bootstrap", "addons.go"))
	if err != nil {
		t.Fatal(err)
	}
	addonsText := string(addonsBody)
	for _, pkg := range []string{
		`"github.com/zatrano/packages/assets"`,
		`"github.com/zatrano/packages/health"`,
		`"github.com/zatrano/packages/localization"`,
		`"github.com/zatrano/packages/view"`,
	} {
		if !strings.Contains(addonsText, pkg) {
			t.Fatalf("web addons.go must blank-import %s:\n%s", pkg, addonsText)
		}
	}
	scaffoldMeta, err := os.ReadFile(filepath.Join(dest, "bootstrap", "scaffold.go"))
	if err != nil {
		t.Fatalf("expected bootstrap/scaffold.go: %v", err)
	}
	metaText := string(scaffoldMeta)
	if !strings.Contains(metaText, `ScaffoldName      = "web"`) || !strings.Contains(metaText, "sha256:") {
		t.Fatalf("scaffold metadata:\n%s", metaText)
	}
	dockerfile, err := os.ReadFile(filepath.Join(dest, "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	df := string(dockerfile)
	if strings.Contains(df, "COPY views ") || strings.Contains(df, "COPY database ") {
		t.Fatalf("Dockerfile still uses legacy top-level copies:\n%s", df)
	}
	if !strings.Contains(df, "COPY app/views") || !strings.Contains(df, "COPY app/database") {
		t.Fatalf("Dockerfile must copy app/views and app/database:\n%s", df)
	}
	if _, err := os.Stat(filepath.Join(dest, "app", "database", "migrations", "20260801_000002_create_jobs_table.go")); err == nil {
		t.Fatal("web starter must not ship queue/jobs migrations")
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
	assertNoPackagesVersionImport(t, dest)
	assertDoctorPass(t, dest)
}

func TestNewEmptyApplication(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "emptyapp")
	cmd := &NewCommand{}
	if err := cmd.Handle([]string{dest, "--module", "example.com/emptyapp", "--replace", root}); err != nil {
		t.Fatal(err)
	}
	assertGeneratedConsoleRegisterABI(t, dest)
	modBytes, err := os.ReadFile(filepath.Join(dest, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	assertGeneratedFrameworkRequire(t, string(modBytes))
	if strings.Contains(string(modBytes), "github.com/zatrano/packages") {
		t.Fatalf("empty go.mod must not pin packages:\n%s", modBytes)
	}
	walk := exec.Command("go", "list", "-deps", "./cmd/app")
	walk.Dir = dest
	out, err := walk.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "github.com/zatrano/packages/") {
		t.Fatalf("empty app has packages deps:\n%s", out)
	}
	enabledBody, err := os.ReadFile(filepath.Join(dest, "bootstrap", "enabled.go"))
	if err != nil {
		t.Fatal(err)
	}
	enabledText := string(enabledBody)
	if strings.Contains(enabledText, `"health"`) || strings.Contains(enabledText, `"view"`) {
		t.Fatalf("empty enabled.go must be empty:\n%s", enabledText)
	}
	meta, err := os.ReadFile(filepath.Join(dest, "bootstrap", "scaffold.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(meta), `ScaffoldName      = "empty"`) {
		t.Fatalf("empty metadata:\n%s", meta)
	}
	home, err := os.ReadFile(filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(home), "http.View") {
		t.Fatalf("empty home must not use view:\n%s", home)
	}
	build := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "emptyapp.exe"), "./cmd/app")
	build.Dir = dest
	if bout, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, bout)
	}
	testCmd := exec.Command("go", "test", "./tests")
	testCmd.Dir = dest
	if testOut, err := testCmd.CombinedOutput(); err != nil {
		t.Fatalf("go test ./tests: %v\n%s", err, testOut)
	}
	assertDoctorPass(t, dest)
}

func TestNewFullApplication(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "fullapp")
	cmd := &NewCommand{}
	if err := cmd.Handle([]string{dest, "--module", "example.com/fullapp", "--replace", root, "--full"}); err != nil {
		t.Fatal(err)
	}
	enabledBody, err := os.ReadFile(filepath.Join(dest, "bootstrap", "enabled.go"))
	if err != nil {
		t.Fatal(err)
	}
	enabledText := string(enabledBody)
	for _, want := range []string{`"assets"`, `"health"`, `"localization"`, `"view"`, `"validation"`} {
		if !strings.Contains(enabledText, want) {
			t.Fatalf("full enabled.go missing %s:\n%s", want, enabledText)
		}
	}
	home, err := os.ReadFile(filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(home), "http.View") {
		t.Fatalf("full home must keep web HTML:\n%s", home)
	}
	meta, err := os.ReadFile(filepath.Join(dest, "bootstrap", "scaffold.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(meta), `ScaffoldName      = "full"`) {
		t.Fatalf("full metadata:\n%s", meta)
	}
	if _, err := os.Stat(filepath.Join(dest, "tests", "api_root_test.go")); err != nil {
		t.Fatal("full must include API root test")
	}
	assertNoPackagesVersionImport(t, dest)
	build := exec.Command("go", "build", "./...")
	build.Dir = dest
	if bout, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, bout)
	}
	testCmd := exec.Command("go", "test", "./tests")
	testCmd.Dir = dest
	if testOut, err := testCmd.CombinedOutput(); err != nil {
		t.Fatalf("go test ./tests: %v\n%s", err, testOut)
	}
	assertDoctorPass(t, dest)
}

func TestNewAPIApplication(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "apiapp")
	cmd := &NewCommand{}
	if err := cmd.Handle([]string{dest, "--module", "example.com/apiapp", "--replace", root, "--api"}); err != nil {
		t.Fatal(err)
	}
	modBytes, err := os.ReadFile(filepath.Join(dest, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	modText := string(modBytes)
	assertGeneratedFrameworkRequire(t, modText)
	if !strings.Contains(modText, "replace github.com/zatrano/packages =>") {
		t.Fatalf("api scaffold must pin packages:\n%s", modText)
	}
	walk := exec.Command("go", "list", "-deps", "./cmd/app")
	walk.Dir = dest
	out, err := walk.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps: %v\n%s", err, out)
	}
	deps := string(out)
	if !strings.Contains(deps, "github.com/zatrano/packages/health") || !strings.Contains(deps, "github.com/zatrano/packages/validation") {
		t.Fatalf("api app must import health and validation:\n%s", deps)
	}
	if strings.Contains(deps, "github.com/zatrano/packages/view") {
		t.Fatalf("api default must not import view:\n%s", deps)
	}
	if strings.Contains(deps, "github.com/zatrano/packages/version") {
		t.Fatalf("api app must not import packages/version:\n%s", deps)
	}
	assertNoPackagesVersionImport(t, dest)
	enabledBody, err := os.ReadFile(filepath.Join(dest, "bootstrap", "enabled.go"))
	if err != nil {
		t.Fatal(err)
	}
	enabledText := string(enabledBody)
	for _, want := range []string{`"health"`, `"validation"`} {
		if !strings.Contains(enabledText, want) {
			t.Fatalf("api enabled.go missing %s:\n%s", want, enabledText)
		}
	}
	if strings.Contains(enabledText, `"view"`) || strings.Contains(enabledText, `"assets"`) {
		t.Fatalf("api enabled.go must not default presentation packages:\n%s", enabledText)
	}
	home, err := os.ReadFile(filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(home), "http.View") {
		t.Fatalf("api home must be JSON, not View:\n%s", home)
	}
	meta, err := os.ReadFile(filepath.Join(dest, "bootstrap", "scaffold.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(meta), `ScaffoldName      = "api"`) {
		t.Fatalf("api scaffold metadata:\n%s", meta)
	}
	if _, err := os.Stat(filepath.Join(dest, "app", "views", "welcome.html")); err != nil {
		t.Fatal("api scaffold must keep view files so package:enable view works")
	}
	build := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "apiapp.exe"), "./cmd/app")
	build.Dir = dest
	bout, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, bout)
	}
	testCmd := exec.Command("go", "test", "./tests")
	testCmd.Dir = dest
	testOut, err := testCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./tests: %v\n%s", err, testOut)
	}
	assertDoctorPass(t, dest)
}

func TestFrameworkGoModVersion(t *testing.T) {
	if got := frameworkGoModVersion("2.0.0"); got != "v2.0.0" {
		t.Fatalf("2.0.0: got %q", got)
	}
	if got := frameworkGoModVersion("2.0.0-dev"); got != "v"+currentRelease {
		t.Fatalf("2.0.0-dev: got %q", got)
	}
	if got := frameworkGoModVersion("1.6.6"); got != "v1.6.6" {
		t.Fatalf("1.6.6: got %q", got)
	}
}

func assertDoctorPass(t *testing.T, dest string) {
	t.Helper()
	findings, err := RunDoctor(dest)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.Severity == "error" {
			t.Fatalf("generated app architecture error %s: %+v\n%s", f.Rule, f, FormatDoctorText(dest, findings))
		}
	}
}

func assertGeneratedFrameworkRequire(t *testing.T, modText string) {
	t.Helper()
	if !strings.Contains(modText, "github.com/zatrano/framework/v2 v"+currentRelease) {
		t.Fatalf("generated require must be v%s:\n%s", currentRelease, modText)
	}
}

func assertNoPackagesVersionImport(t *testing.T, dest string) {
	t.Helper()
	home := filepath.Join(dest, "app", "http", "controllers", "api", "home_controller.go")
	body, err := os.ReadFile(home)
	if err != nil {
		t.Fatalf("expected %s: %v", home, err)
	}
	if strings.Contains(string(body), "github.com/zatrano/packages/version") {
		t.Fatalf("generated API home must not import packages/version:\n%s", body)
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
	if got := generator.StripTmplSuffix("go.mod.tmpl"); got != "go.mod" {
		t.Fatalf("got %q", got)
	}
	if got := generator.StripTmplSuffix("cmd/app/main.go.tmpl"); got != "cmd/app/main.go" {
		t.Fatalf("got %q", got)
	}
}

func TestEmbeddedScaffoldDigestsAreDeterministicAndDistinct(t *testing.T) {
	web1, err := generator.Digest(starterTemplates, "templates/web")
	if err != nil {
		t.Fatal(err)
	}
	web2, err := generator.Digest(starterTemplates, "templates/web")
	if err != nil {
		t.Fatal(err)
	}
	empty1, err := generator.Digest(starterTemplates, "templates/empty")
	if err != nil {
		t.Fatal(err)
	}
	empty2, err := generator.Digest(starterTemplates, "templates/empty")
	if err != nil {
		t.Fatal(err)
	}
	api1, err := generator.Digest(starterTemplates, "templates/api")
	if err != nil {
		t.Fatal(err)
	}
	api2, err := generator.Digest(starterTemplates, "templates/api")
	if err != nil {
		t.Fatal(err)
	}
	if web1 != web2 || empty1 != empty2 || api1 != api2 {
		t.Fatalf("digest not stable web=%s/%s empty=%s/%s api=%s/%s", web1, web2, empty1, empty2, api1, api2)
	}
	if web1 == empty1 || web1 == api1 || empty1 == api1 {
		t.Fatal("web, api, and empty scaffolds must have distinct digests")
	}
	if !strings.HasPrefix(web1, "sha256:") || !strings.HasPrefix(empty1, "sha256:") || !strings.HasPrefix(api1, "sha256:") {
		t.Fatalf("digest scheme web=%s empty=%s api=%s", web1, empty1, api1)
	}
}
