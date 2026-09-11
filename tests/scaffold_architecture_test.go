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

	if st, err := os.Stat(filepath.Join(root, "console", "stubs")); err == nil && st.IsDir() {
		t.Fatal("console/stubs must not exist; auth/dashboard stubs belong in github.com/zatrano/packages/auth")
	}
	if st, err := os.Stat(filepath.Join(root, "bootstrap", "stubs")); err == nil && st.IsDir() {
		t.Fatal("bootstrap/stubs must not exist; package config bodies belong on addons.Meta.ConfigFiles")
	}
	if _, err := os.Stat(filepath.Join(root, "console", "request.go")); err == nil {
		t.Fatal("make:request must not live in the framework console")
	}
	if _, err := os.Stat(filepath.Join(root, "console", "rule.go")); err == nil {
		t.Fatal("make:rule must not live in the framework console")
	}

	if _, err := os.Stat(filepath.Join(root, "console", "add.go")); err == nil {
		t.Fatal("add:web / add:api must not exist")
	}

	entries, err := os.ReadDir(filepath.Join(root, "console", "templates"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			t.Errorf("console/templates must contain only scaffold directories, found file %s", e.Name())
			continue
		}
		switch e.Name() {
		case "web":
		default:
			t.Errorf("unexpected scaffold directory console/templates/%s", e.Name())
		}
	}

	doctorSrc, err := os.ReadFile(filepath.Join(root, "console", "doctor_checks.go"))
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

func TestStarterEnablementImportsPresentationPackages(t *testing.T) {
	root := filepath.Join(moduleRoot(t), "console", "templates", "web")
	addons, err := os.ReadFile(filepath.Join(root, "bootstrap", "addons.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(addons)
	for _, pkg := range []string{
		`"github.com/zatrano/packages/assets"`,
		`"github.com/zatrano/packages/health"`,
		`"github.com/zatrano/packages/localization"`,
		`"github.com/zatrano/packages/validation"`,
		`"github.com/zatrano/packages/view"`,
	} {
		if !strings.Contains(text, pkg) {
			t.Errorf("starter addons.go.tmpl must blank-import %s", pkg)
		}
	}
}

func TestEmbeddedDockerfilesMatchCanonicalLayout(t *testing.T) {
	root := moduleRoot(t)
	for _, rel := range []string{
		filepath.Join("console", "templates", "web", "Dockerfile"),
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
	body, err := os.ReadFile(filepath.Join(root, "console", "templates", "web", "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "COPY app/views") || !strings.Contains(text, "COPY app/database") {
		t.Fatal("web Dockerfile must copy app/views and app/database")
	}
}

func TestWebScaffoldDoesNotShipPackageMigrations(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "console", "templates", "web", "app", "database", "migrations")
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if strings.Contains(name, "job") || strings.Contains(name, "notification") {
			t.Errorf("starter must not ship package migration %s", d.Name())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGeneratorEngineHasNoApplicationPolicy(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(moduleRoot(t), "console", "generator", "engine.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	for _, ban := range []string{"oauth", "welcome.html", "make:auth", "github.com/zatrano/packages"} {
		if strings.Contains(text, ban) {
			t.Errorf("generator engine contains application policy %q", ban)
		}
	}
	newSrc, err := os.ReadFile(filepath.Join(moduleRoot(t), "console", "new.go"))
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
	webRoot := filepath.Join(root, "console", "templates", "web")
	if len(relFiles(t, webRoot)) == 0 {
		t.Fatal("web starter inventory")
	}
	enabled, err := os.ReadFile(filepath.Join(webRoot, "bootstrap", "enabled.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(enabled)
	for _, want := range []string{`"assets"`, `"health"`, `"localization"`, `"validation"`, `"view"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("starter enablement missing %s:\n%s", want, text)
		}
	}
	webHome, err := os.ReadFile(filepath.Join(webRoot, "app", "http", "controllers", "web", "home_controller.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(webHome), "http.View") {
		t.Fatal("starter web home must be HTML View")
	}
	apiHome, err := os.ReadFile(filepath.Join(webRoot, "app", "http", "controllers", "api", "home_controller.go.tmpl"))
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
