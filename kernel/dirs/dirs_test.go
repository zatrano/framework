package dirs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestDirPrefersNewTree(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	if err := os.MkdirAll(filepath.Join(dir, "app", "views"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "views"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := ViewsDir(app)
	want := filepath.Join(dir, "app", "views")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDirFallsBack(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	if err := os.MkdirAll(filepath.Join(dir, "lang"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := LocalizationDir(app)
	want := filepath.Join(dir, "lang")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDirForCreateUsesNewWhenMissing(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	got := DatabaseDirForCreate(app)
	want := filepath.Join(dir, "app", "database")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestHTTPSurfaces(t *testing.T) {
	if !RouteRelAllowed("app/routes/auth/web/auth.go") || !ControllerRelAllowed("app/http/controllers/auth/web/auth_controller.go") {
		t.Fatal("auth is a built-in HTTP surface")
	}
	if !RouteRelAllowed("app/routes/auth/api/auth.go") || !ControllerRelAllowed("app/http/controllers/auth/api/auth_controller.go") {
		t.Fatal("auth api nest is a built-in HTTP surface")
	}
	if ControllerKind("app/http/controllers/auth/api/auth_controller.go") != SurfaceAPI {
		t.Fatal("auth/api controllers are JSON")
	}
	if ControllerKind("app/http/controllers/auth/web/auth_controller.go") != SurfaceWeb {
		t.Fatal("auth/web controllers are HTML")
	}
	if !RouteRelAllowed("app/routes/dashboard/home.go") || !ValidPanelName("dashboard") {
		t.Fatal("make:panel names are route/controller surfaces")
	}
	if ValidPanelName("web") || ValidPanelName("api") || ValidPanelName("auth") || ValidPanelName("1ops") {
		t.Fatal("reserved or non-canonical panel names")
	}
	if RouteRelAllowed("app/http/routes/web.go") || ControllerRelAllowed("app/http/controllers/home.go") {
		t.Fatal("non-surface paths must be rejected")
	}
}

func TestCanonicalConsumerDirsArePlatformNotWeb(t *testing.T) {
	got := CanonicalConsumerDirs()
	if len(got) == 0 {
		t.Fatal("canonical dirs empty")
	}
	join := strings.Join(got, "\n")
	for _, d := range []string{"app/providers", "app/routes/web", "cmd/app", "bootstrap"} {
		if !strings.Contains(join, d) {
			t.Fatalf("missing canonical %s in %v", d, got)
		}
	}
	for _, d := range OptionalWebScaffoldDirs() {
		if strings.Contains(join, d) {
			t.Fatalf("web-only dir %s must not be canonical", d)
		}
	}
}

func TestDirNilAppAndCreateHelpers(t *testing.T) {
	if got := Dir(nil, []string{"a"}, []string{"b", "c"}); got == "" {
		t.Fatal("nil Dir")
	}
	if got := DirForCreate(nil, []string{"pref"}, []string{"fb"}); !strings.Contains(got, "pref") {
		t.Fatalf("nil DirForCreate %q", got)
	}
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	if ViewsDirForCreate(app) != filepath.Join(dir, "app", "views") {
		t.Fatal("ViewsDirForCreate")
	}
	if LocalizationDirForCreate(app) != filepath.Join(dir, "app", "localization") {
		t.Fatal("LocalizationDirForCreate")
	}
	if DatabaseDir(app) != filepath.Join(dir, "database") && DatabaseDir(app) != filepath.Join(dir, "app", "database") {
		// neither dir exists; Dir falls back to fallback path
		if DatabaseDir(app) != filepath.Join(dir, "database") {
			t.Fatalf("DatabaseDir=%s", DatabaseDir(app))
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "app", "database"), 0o755); err != nil {
		t.Fatal(err)
	}
	if DatabaseDir(app) != filepath.Join(dir, "app", "database") {
		t.Fatal("DatabaseDir preferred")
	}
	routes := CanonicalRouteDirs()
	if len(routes) != 2 {
		t.Fatalf("%v", routes)
	}
	if err := os.MkdirAll(filepath.Join(dir, "views"), 0o755); err != nil {
		t.Fatal(err)
	}
	if ViewsDirForCreate(app) != filepath.Join(dir, "views") {
		t.Fatalf("create prefers existing fallback: %s", ViewsDirForCreate(app))
	}
}
