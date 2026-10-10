package tests

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestScaffoldBoundaries(t *testing.T) {
	root := moduleRoot(t)

	if st, err := os.Stat(filepath.Join(root, "core", "console", "stubs")); err == nil && st.IsDir() {
		t.Fatal("core/console/stubs must not exist; auth/dashboard stubs belong in github.com/zatrano/packages/auth")
	}
	if st, err := os.Stat(filepath.Join(root, "core", "bootstrap", "stubs")); err == nil && st.IsDir() {
		t.Fatal("core/bootstrap/stubs must not exist; package config bodies belong on addons.Meta.ConfigFiles")
	}
	if _, err := os.Stat(filepath.Join(root, "core", "console", "request.go")); err == nil {
		t.Fatal("make:request must not live in the framework console")
	}
	if _, err := os.Stat(filepath.Join(root, "core", "console", "rule.go")); err == nil {
		t.Fatal("make:rule must not live in the framework console")
	}

	if _, err := os.Stat(filepath.Join(root, "core", "console", "add.go")); err == nil {
		t.Fatal("add:web / add:api must not exist")
	}

	entries, err := os.ReadDir(filepath.Join(root, "core", "console", "scaffold", "templates"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			t.Errorf("core/console/templates must contain only scaffold directories, found file %s", e.Name())
			continue
		}
		switch e.Name() {
		case "web":
		default:
			t.Errorf("unexpected scaffold directory console/templates/%s", e.Name())
		}
	}

	doctorSrc, err := os.ReadFile(filepath.Join(root, "core", "console", "doctor", "doctor_checks.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(doctorSrc)
	if !strings.Contains(text, "dirs.CanonicalConsumerDirs()") {
		t.Fatal("doctor must use kernel/dirs.CanonicalConsumerDirs")
	}
	if strings.Contains(text, "WalkDir") && strings.Contains(text, "templates") {
		t.Fatal("doctor must not derive canonical layout by walking starter templates")
	}
}

func TestStarterEnablementImportsHealthAndCanvasSSR(t *testing.T) {
	root := filepath.Join(moduleRoot(t), "core", "console", "scaffold", "templates", "web")
	addons, err := os.ReadFile(filepath.Join(root, "bootstrap", "addons.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(addons)
	for _, pkg := range []string{
		`"github.com/zatrano/packages/health"`,
		`"github.com/zatrano/framework/v3/core/ssr"`,
	} {
		if !strings.Contains(text, pkg) {
			t.Errorf("starter addons.go.tmpl must blank-import %s", pkg)
		}
	}
	for _, pkg := range []string{
		`"github.com/zatrano/packages/assets"`,
		`"github.com/zatrano/packages/localization"`,
		`"github.com/zatrano/packages/validation"`,
	} {
		if strings.Contains(text, pkg) {
			t.Errorf("starter addons.go.tmpl must not default-import %s", pkg)
		}
	}
}

func TestEmbeddedDockerfilesMatchCanonicalLayout(t *testing.T) {
	root := moduleRoot(t)
	for _, rel := range []string{
		filepath.Join("core", "console", "scaffold", "templates", "web", "Dockerfile"),
	} {
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		if strings.Contains(text, "COPY views ") || strings.Contains(text, "COPY database ") {
			t.Errorf("%s uses legacy top-level COPY", filepath.ToSlash(rel))
		}
	}
	body, err := os.ReadFile(filepath.Join(root, "core", "console", "scaffold", "templates", "web", "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "COPY templates") {
		t.Fatal("web Dockerfile must copy Canvas templates")
	}
	if !strings.Contains(text, "FROM golang:1.26-alpine") || !strings.Contains(text, "FROM alpine:3.24") {
		t.Fatal("web Dockerfile must build with golang:1.26-alpine and run on alpine:3.24")
	}
	mod, err := os.ReadFile(filepath.Join(root, "core", "console", "scaffold", "templates", "web", "go.mod.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mod), "\ntoolchain ") || strings.HasPrefix(string(mod), "toolchain ") {
		t.Fatal("generated go.mod must not set a toolchain line")
	}
	if strings.Contains(text, "COPY app/database") {
		t.Fatal("web Dockerfile must not copy opt-in app/database")
	}
}

func TestWebScaffoldDoesNotShipPackageMigrations(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "core", "console", "scaffold", "templates", "web", "app", "database")
	if _, err := os.Stat(dir); err == nil {
		t.Fatal("starter must not ship app/database")
	}
}

func TestGeneratorEngineHasNoApplicationPolicy(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(moduleRoot(t), "core", "console", "generator", "engine.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	for _, ban := range []string{"oauth", "welcome.html", "make:auth", "github.com/zatrano/packages"} {
		if strings.Contains(text, ban) {
			t.Errorf("generator engine contains application policy %q", ban)
		}
	}
	newSrc, err := os.ReadFile(filepath.Join(moduleRoot(t), "core", "console", "scaffold", "new.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(newSrc), "generator.Apply") {
		t.Fatal("zatrano new must use the generator engine")
	}
	if strings.Contains(string(newSrc), "applyMinimalScaffold") {
		t.Fatal("empty must not use a second string-literal generation path")
	}
}

func TestStarterHasWebAndAPIPresentation(t *testing.T) {
	root := moduleRoot(t)
	webRoot := filepath.Join(root, "core", "console", "scaffold", "templates", "web")
	if len(relFiles(t, webRoot)) == 0 {
		t.Fatal("web starter inventory")
	}
	enabled, err := os.ReadFile(filepath.Join(webRoot, "bootstrap", "enabled.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(enabled)
	for _, want := range []string{`"health"`, `"template"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("starter enablement missing %s:\n%s", want, text)
		}
	}
	for _, deny := range []string{`"assets"`, `"localization"`, `"validation"`} {
		if strings.Contains(text, deny) {
			t.Fatalf("starter must not default-enable %s:\n%s", deny, text)
		}
	}
	webHome, err := os.ReadFile(filepath.Join(webRoot, "app", "http", "handlers", "web", "home_handler.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(webHome), `http.Template("web.welcome")`) {
		t.Fatal("starter web home must use Canvas Template")
	}
	if strings.Contains(string(webHome), "http.HTML") {
		t.Fatal("starter web home must not use raw http.HTML")
	}
	apiHome, err := os.ReadFile(filepath.Join(webRoot, "app", "http", "handlers", "api", "home_handler.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(apiHome), "http.JSON") {
		t.Fatal("starter API home must be JSON")
	}
	if _, err := os.Stat(filepath.Join(webRoot, "tests", "api_root_test.go.tmpl")); err != nil {
		t.Fatal("starter must include API root test")
	}
}

func relFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

func TestStarterSmokeUsesLiveAddon(t *testing.T) {
	root := moduleRoot(t)
	body, err := os.ReadFile(filepath.Join(root, ".github", "scripts", "starter-smoke.sh"))
	if err != nil {
		t.Fatal(err)
	}
	src := strings.ReplaceAll(string(body), "\r\n", "\n")
	if !strings.Contains(src, `github.com/zatrano/packages/audit`) {
		t.Fatal("starter-smoke must blank-import a live addon (audit)")
	}
	if !strings.Contains(src, "\n    qr\n") {
		t.Fatal("starter-smoke nested replaces must include qr")
	}
	for _, gone := range []string{"packages/billing", "packages/octane", "packages/bus", "packages/features", "packages/tenancy"} {
		if strings.Contains(src, gone) {
			t.Fatalf("starter-smoke must not import removed package %s", gone)
		}
	}
}
