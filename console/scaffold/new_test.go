package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/console/consolecore"
	"github.com/zatrano/framework/v2/console/doctor"
	"github.com/zatrano/framework/v2/console/generator"
)

func TestNewSeedsDotEnv(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "seeded")
	if err := (&NewCommand{}).Handle([]string{dest}); err != nil {
		t.Fatal(err)
	}
	assertSeededEnv(t, dest)
}

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

func TestNewHelpHasNoProfileFlags(t *testing.T) {
	if _, _, _, err := parseNewArgs([]string{"demo", "--unknown"}); err == nil || !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("expected unknown flag, got %v", err)
	}
	for _, name := range []string{"--web", "--full", "--empty", "--minimal", "add:web", "add:api"} {
		if strings.Contains(newHelp, name) {
			t.Fatalf("new help must not mention %s:\n%s", name, newHelp)
		}
	}
}

func TestAddCommandsRemoved(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "add.go")); err == nil {
		t.Fatal("add.go must not exist")
	}
	src, err := os.ReadFile(filepath.Join("..", "console.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	if strings.Contains(text, "registerAddCommands") || strings.Contains(text, "add:web") || strings.Contains(text, "add:api") {
		t.Fatal("console must not register add:web / add:api")
	}
}

func TestG001UpgradeDoesNotRegenerateApplicationSource(t *testing.T) {
	upgrade, err := os.ReadFile(filepath.Join("..", "..", "tests", "compatibility", "upgrade_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(upgrade)
	if strings.Contains(text, "add:web") || strings.Contains(text, "add:api") {
		t.Fatal("G-001 upgrade must not invoke add:web/add:api")
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

func TestNewApplication(t *testing.T) {
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
	if _, err := os.Stat(filepath.Join(dest, "app", "views")); err == nil {
		t.Fatal("starter must not ship app/views until package:enable view")
	}
	assertSeededEnv(t, dest)
	if _, err := os.Stat(filepath.Join(dest, "app", "database")); err == nil {
		t.Fatal("starter must not ship app/database until package:enable database")
	}
	if _, err := os.Stat(filepath.Join(dest, "app", "localization")); err == nil {
		t.Fatal("starter must not ship app/localization until package:enable localization / lang:publish")
	}
	if _, err := os.Stat(filepath.Join(dest, "config", ".gitkeep")); err != nil {
		t.Fatalf("expected minimal config/: %v", err)
	}
	assertStarterAppDirBudget(t, dest)
	if _, err := os.Stat(filepath.Join(dest, "bootstrap", "addons.go")); err != nil {
		t.Fatalf("expected bootstrap/addons.go: %v", err)
	}
	addonsBody, err := os.ReadFile(filepath.Join(dest, "bootstrap", "addons.go"))
	if err != nil {
		t.Fatal(err)
	}
	addonsText := string(addonsBody)
	for _, pkg := range []string{
		`"github.com/zatrano/packages/health"`,
	} {
		if !strings.Contains(addonsText, pkg) {
			t.Fatalf("web addons.go must blank-import %s:\n%s", pkg, addonsText)
		}
	}
	for _, pkg := range []string{
		`"github.com/zatrano/packages/assets"`,
		`"github.com/zatrano/packages/localization"`,
		`"github.com/zatrano/packages/view"`,
		`"github.com/zatrano/packages/validation"`,
	} {
		if strings.Contains(addonsText, pkg) {
			t.Fatalf("web addons.go must not default-import opt-in %s:\n%s", pkg, addonsText)
		}
	}
	scaffoldMeta, err := os.ReadFile(filepath.Join(dest, "bootstrap", "scaffold.go"))
	if err != nil {
		t.Fatalf("expected bootstrap/scaffold.go: %v", err)
	}
	metaText := string(scaffoldMeta)
	if !strings.Contains(metaText, `ScaffoldName      = "app"`) || !strings.Contains(metaText, "sha256:") {
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
	if strings.Contains(df, "COPY app/views") {
		t.Fatalf("Dockerfile must not copy opt-in app/views:\n%s", df)
	}
	if strings.Contains(df, "COPY app/database") {
		t.Fatalf("Dockerfile must not copy opt-in app/database:\n%s", df)
	}
	assertGeneratedMainHasNoAppConsole(t, dest)
	enabledBody, err := os.ReadFile(filepath.Join(dest, "bootstrap", "enabled.go"))
	if err != nil {
		t.Fatalf("expected bootstrap/enabled.go: %v", err)
	}
	enabledText := string(enabledBody)
	for _, want := range []string{"RegisterEnablement", `"health"`} {
		if !strings.Contains(enabledText, want) {
			t.Fatalf("enabled.go missing %q:\n%s", want, enabledText)
		}
	}
	for _, deny := range []string{`"assets"`, `"localization"`, `"view"`, `"validation"`} {
		if strings.Contains(enabledText, deny) {
			t.Fatalf("enabled.go must not default-enable %s:\n%s", deny, enabledText)
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
	if _, err := os.Stat(filepath.Join(dest, "tests", "api_root_test.go")); err != nil {
		t.Fatal("generated app must include API root test")
	}
	home, err := os.ReadFile(filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(home), "http.HTML") {
		t.Fatalf("generated home must be kernel HTML:\n%s", home)
	}
	if strings.Contains(string(home), "http.View") {
		t.Fatalf("generated home must not use view until package:enable view:\n%s", home)
	}
	assertNoPackagesVersionImport(t, dest)
	assertDoctorPass(t, dest)
	assertMakeJobCreatesAppJobs(t, dest)
}

func TestFrameworkGoModVersion(t *testing.T) {
	if got := frameworkGoModVersion("2.0.0"); got != "v2.0.0" {
		t.Fatalf("2.0.0: got %q", got)
	}
	if got := frameworkGoModVersion("2.0.0-dev"); got != "v"+consolecore.CurrentRelease {
		t.Fatalf("2.0.0-dev: got %q", got)
	}
	if got := frameworkGoModVersion("1.6.6"); got != "v1.6.6" {
		t.Fatalf("1.6.6: got %q", got)
	}
}

func assertSeededEnv(t *testing.T, dest string) {
	t.Helper()
	example, err := os.ReadFile(filepath.Join(dest, ".env.example"))
	if err != nil {
		t.Fatalf("expected .env.example: %v", err)
	}
	env, err := os.ReadFile(filepath.Join(dest, ".env"))
	if err != nil {
		t.Fatalf("zatrano new must seed .env from .env.example: %v", err)
	}
	if string(env) != string(example) {
		t.Fatalf("seeded .env must match .env.example\n.env:\n%s\n.env.example:\n%s", env, example)
	}
}

func assertDoctorPass(t *testing.T, dest string) {
	t.Helper()
	findings, err := doctor.RunDoctor(dest)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.Severity == "error" {
			t.Fatalf("generated app architecture error %s: %+v\n%s", f.Rule, f, doctor.FormatDoctorText(dest, findings))
		}
	}
}

func assertGeneratedFrameworkRequire(t *testing.T, modText string) {
	t.Helper()
	if !strings.Contains(modText, "github.com/zatrano/framework/v2 v"+consolecore.CurrentRelease) {
		t.Fatalf("generated require must be v%s:\n%s", consolecore.CurrentRelease, modText)
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

func assertStarterAppDirBudget(t *testing.T, dest string) {
	t.Helper()
	appDir := filepath.Join(dest, "app")
	entries, err := os.ReadDir(appDir)
	if err != nil {
		t.Fatal(err)
	}
	allow := map[string]bool{"http": true, "providers": true, "routes": true}
	n := 1
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n++
		if !allow[e.Name()] {
			t.Fatalf("starter app/%s is not a default-enabled package directory", e.Name())
		}
	}
	if n > 5 {
		t.Fatalf("find %s -maxdepth 1 -type d counted %d, want <= 5", appDir, n)
	}
}

func assertGeneratedMainHasNoAppConsole(t *testing.T, dest string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(dest, "cmd", "app", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "/app/console") || strings.Contains(text, "console.Register(") {
		t.Fatalf("starter main.go must not import app/console until make:command:\n%s", text)
	}
}

func assertMakeJobCreatesAppJobs(t *testing.T, dest string) {
	t.Helper()
	enable := exec.Command("go", "run", "./cmd/app", "package:enable", "queue")
	enable.Dir = dest
	out, err := enable.CombinedOutput()
	if err != nil {
		t.Fatalf("package:enable queue: %v\n%s", err, out)
	}
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dest
	out, err = tidy.CombinedOutput()
	if err != nil {
		t.Fatalf("go mod tidy after package:enable queue: %v\n%s", err, out)
	}
	job := exec.Command("go", "run", "./cmd/app", "make:job", "SendWelcome")
	job.Dir = dest
	out, err = job.CombinedOutput()
	if err != nil {
		t.Fatalf("make:job: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(dest, "app", "jobs", "send_welcome.go")); err != nil {
		t.Fatalf("make:job must create app/jobs file: %v\n%s", err, out)
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

func TestEmbeddedScaffoldDigestIsDeterministic(t *testing.T) {
	web1, err := generator.Digest(starterTemplates, "templates/web")
	if err != nil {
		t.Fatal(err)
	}
	web2, err := generator.Digest(starterTemplates, "templates/web")
	if err != nil {
		t.Fatal(err)
	}
	if web1 != web2 {
		t.Fatalf("digest not stable %s/%s", web1, web2)
	}
	if !strings.HasPrefix(web1, "sha256:") {
		t.Fatalf("digest scheme %s", web1)
	}
}
