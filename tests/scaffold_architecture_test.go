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
		case "empty", "web", "api", "overlays":
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

func TestEmptyScaffoldHasNoPackageImports(t *testing.T) {
	root := filepath.Join(moduleRoot(t), "console", "templates", "empty")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		switch {
		case strings.HasSuffix(name, ".go"), strings.HasSuffix(name, ".tmpl"), name == "go.mod.tmpl", name == "go.mod":
		default:
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(body), "\n") {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "//") {
				continue
			}
			if strings.Contains(trim, `"github.com/zatrano/packages`) {
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s imports packages: %s", filepath.ToSlash(rel), trim)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEmbeddedDockerfilesMatchCanonicalLayout(t *testing.T) {
	root := moduleRoot(t)
	for _, rel := range []string{
		filepath.Join("console", "templates", "web", "Dockerfile"),
		filepath.Join("console", "templates", "api", "Dockerfile"),
		filepath.Join("console", "templates", "empty", "Dockerfile"),
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
	for _, scaffold := range []string{"web", "api"} {
		body, err := os.ReadFile(filepath.Join(root, "console", "templates", scaffold, "Dockerfile"))
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		if !strings.Contains(text, "COPY app/views") || !strings.Contains(text, "COPY app/database") {
			t.Fatalf("%s Dockerfile must copy app/views and app/database", scaffold)
		}
	}
}

func TestWebScaffoldDoesNotShipPackageMigrations(t *testing.T) {
	root := moduleRoot(t)
	for _, scaffold := range []string{"web", "api"} {
		dir := filepath.Join(root, "console", "templates", scaffold, "app", "database", "migrations")
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			name := strings.ToLower(d.Name())
			if strings.Contains(name, "job") || strings.Contains(name, "notification") {
				t.Errorf("%s scaffold must not ship package migration %s", scaffold, d.Name())
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
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

func TestEmptyAndWebShareLayoutDifferInPresentation(t *testing.T) {
	root := moduleRoot(t)
	emptyRoot := filepath.Join(root, "console", "templates", "empty")
	webRoot := filepath.Join(root, "console", "templates", "web")
	emptyPaths := relFiles(t, emptyRoot)
	webPaths := relFiles(t, webRoot)
	if strings.Join(emptyPaths, "\n") != strings.Join(webPaths, "\n") {
		t.Fatalf("path inventory drifted\nempty=%d web=%d", len(emptyPaths), len(webPaths))
	}
	if len(emptyPaths) == 0 {
		t.Fatal("empty scaffold inventory")
	}
	policy := []string{
		"Dockerfile",
		"app/database/migrations/migrations.go.tmpl",
		"app/database/seeders/database_seeder.go.tmpl",
		"app/http/controllers/api/home_controller.go.tmpl",
		"app/http/controllers/web/home_controller.go.tmpl",
		"app/providers/app_service_provider.go.tmpl",
		"app/providers/providers.go.tmpl",
		"app/routes/web/health.go.tmpl",
		"app/routes/web/web.go.tmpl",
		"bootstrap/enabled.go.tmpl",
		"tests/feature_test.go.tmpl",
	}
	for _, rel := range policy {
		eb, err := os.ReadFile(filepath.Join(emptyRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		wb, err := os.ReadFile(filepath.Join(webRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if string(eb) == string(wb) {
			t.Errorf("policy file %s must differ between empty and web", rel)
		}
	}
	home, err := os.ReadFile(filepath.Join(emptyRoot, "app", "http", "controllers", "web", "home_controller.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(home), "http.View") || strings.Contains(string(home), "github.com/zatrano/packages") {
		t.Fatalf("empty home controller must stay kernel HTML:\n%s", home)
	}
	enabled, err := os.ReadFile(filepath.Join(emptyRoot, "bootstrap", "enabled.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(enabled), `"view"`) || strings.Contains(string(enabled), `"health"`) {
		t.Fatalf("empty enablement must be empty:\n%s", enabled)
	}
}

func TestAPIAndWebShareLayoutDifferInPresentation(t *testing.T) {
	root := moduleRoot(t)
	apiRoot := filepath.Join(root, "console", "templates", "api")
	webRoot := filepath.Join(root, "console", "templates", "web")
	if strings.Join(relFiles(t, apiRoot), "\n") != strings.Join(relFiles(t, webRoot), "\n") {
		t.Fatal("api and web must share the same layout inventory so either can enable any package")
	}
	apiEnabled, err := os.ReadFile(filepath.Join(apiRoot, "bootstrap", "enabled.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(apiEnabled)
	if !strings.Contains(text, `"health"`) || !strings.Contains(text, `"validation"`) {
		t.Fatalf("api defaults: %s", text)
	}
	if strings.Contains(text, `"view"`) {
		t.Fatal("api must not default-enable view")
	}
	webEnabled, err := os.ReadFile(filepath.Join(webRoot, "bootstrap", "enabled.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(webEnabled), `"view"`) {
		t.Fatal("web must default-enable view")
	}
	apiHome, err := os.ReadFile(filepath.Join(apiRoot, "app", "http", "controllers", "web", "home_controller.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(apiHome), "http.View") {
		t.Fatal("api root must be JSON")
	}
	mig, err := os.ReadFile(filepath.Join(apiRoot, "app", "database", "migrations", "migrations.go.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mig), "github.com/zatrano/packages/database/migration") {
		t.Fatal("api must use typed migrations so package:enable database is first-class")
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
