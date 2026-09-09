package console

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func generateApp(t *testing.T, name string, flags ...string) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), name)
	args := append([]string{dest, "--module", "example.com/" + name, "--replace", root}, flags...)
	if err := (&NewCommand{}).Handle(args); err != nil {
		t.Fatal(err)
	}
	return dest
}

func TestAddAPIToWebApplication(t *testing.T) {
	dest := generateApp(t, "webplus", "--web")
	homeBefore, _ := os.ReadFile(filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go"))
	metaBefore, _ := os.ReadFile(filepath.Join(dest, "bootstrap", "scaffold.go"))
	app := kernel.NewApplication(dest)
	if err := (&AddAPICommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	homeAfter, _ := os.ReadFile(filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go"))
	if string(homeBefore) != string(homeAfter) {
		t.Fatal("add:api must not overwrite web home controller")
	}
	metaAfter, _ := os.ReadFile(filepath.Join(dest, "bootstrap", "scaffold.go"))
	if string(metaBefore) != string(metaAfter) {
		t.Fatal("add:api must not rewrite scaffold metadata")
	}
	enabled, _ := os.ReadFile(filepath.Join(dest, "bootstrap", "enabled.go"))
	text := string(enabled)
	if !strings.Contains(text, `"view"`) || !strings.Contains(text, `"validation"`) {
		t.Fatalf("web+api enablement:\n%s", text)
	}
	if _, err := os.Stat(filepath.Join(dest, "tests", "api_root_test.go")); err != nil {
		t.Fatal("add:api must write API root test")
	}
	buildAndTest(t, dest)
}

func TestAddWebToAPIApplication(t *testing.T) {
	dest := generateApp(t, "apiplus", "--api")
	homeBefore, _ := os.ReadFile(filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go"))
	app := kernel.NewApplication(dest)
	if err := (&AddWebCommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	homeAfter, _ := os.ReadFile(filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go"))
	if string(homeBefore) != string(homeAfter) {
		t.Fatal("add:web must not overwrite API home controller")
	}
	if !strings.Contains(string(homeAfter), "http.JSON") {
		t.Fatal("API JSON / must remain")
	}
	enabled, _ := os.ReadFile(filepath.Join(dest, "bootstrap", "enabled.go"))
	text := string(enabled)
	if !strings.Contains(text, `"view"`) || !strings.Contains(text, `"validation"`) {
		t.Fatalf("api+web enablement:\n%s", text)
	}
	if _, err := os.Stat(filepath.Join(dest, "app", "views", "welcome.html")); err != nil {
		t.Fatal("web views must remain available")
	}
	buildAndTest(t, dest)
}

func TestAddWebIdempotent(t *testing.T) {
	dest := generateApp(t, "webid", "--web")
	app := kernel.NewApplication(dest)
	if err := (&AddWebCommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	snap := snapshotPresentation(t, dest)
	if err := (&AddWebCommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	assertSnapshotUnchanged(t, dest, snap)
}

func TestAddAPIIdempotent(t *testing.T) {
	dest := generateApp(t, "apiid", "--api")
	app := kernel.NewApplication(dest)
	if err := (&AddAPICommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	snap := snapshotPresentation(t, dest)
	if err := (&AddAPICommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	assertSnapshotUnchanged(t, dest, snap)
	buildAndTest(t, dest)
}

func TestFullEquivalentToWebPlusAPI(t *testing.T) {
	full := generateApp(t, "fulleq", "--full")
	web := generateApp(t, "webeq", "--web")
	if err := (&AddAPICommand{app: kernel.NewApplication(web)}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	fullEn := parseEnabledAddons(string(mustRead(t, filepath.Join(full, "bootstrap", "enabled.go"))))
	webEn := parseEnabledAddons(string(mustRead(t, filepath.Join(web, "bootstrap", "enabled.go"))))
	if strings.Join(fullEn, ",") != strings.Join(webEn, ",") {
		t.Fatalf("enablement full=%v web+api=%v", fullEn, webEn)
	}
	fullHome := string(mustRead(t, filepath.Join(full, "app", "http", "controllers", "web", "home_controller.go")))
	webHome := string(mustRead(t, filepath.Join(web, "app", "http", "controllers", "web", "home_controller.go")))
	if !strings.Contains(fullHome, "http.View") || !strings.Contains(webHome, "http.View") {
		t.Fatal("both must keep HTML /")
	}
	buildAndTest(t, full)
	buildAndTest(t, web)
}

func TestFullCapabilityEquivalentToWebPlusAPI(t *testing.T) {
	TestFullEquivalentToWebPlusAPI(t)
}

func TestAddAPIDoesNotChangeWebRootBehavior(t *testing.T) {
	dest := generateApp(t, "webroot", "--web")
	home := string(mustRead(t, filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go")))
	if !strings.Contains(home, "http.View") {
		t.Fatal("web root must be HTML")
	}
	if err := (&AddAPICommand{app: kernel.NewApplication(dest)}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	homeAfter := string(mustRead(t, filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go")))
	if home != homeAfter {
		t.Fatal("add:api must preserve web root source")
	}
	if !strings.Contains(homeAfter, "http.View") {
		t.Fatal("web HTML / must remain after add:api")
	}
	routes := string(mustRead(t, filepath.Join(dest, "app", "routes", "web", "web.go")))
	if !strings.Contains(routes, `Get("/", c.Index)`) {
		t.Fatal("web routes must remain")
	}
	buildAndTest(t, dest)
}

func TestAddWebDoesNotChangeAPIRootBehavior(t *testing.T) {
	dest := generateApp(t, "apiroot", "--api")
	home := string(mustRead(t, filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go")))
	if !strings.Contains(home, "http.JSON") {
		t.Fatal("api root must be JSON")
	}
	apiRoutes := string(mustRead(t, filepath.Join(dest, "app", "routes", "api", "api.go")))
	if err := (&AddWebCommand{app: kernel.NewApplication(dest)}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	homeAfter := string(mustRead(t, filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go")))
	if home != homeAfter || !strings.Contains(homeAfter, "http.JSON") {
		t.Fatal("add:web must preserve API JSON /")
	}
	apiAfter := string(mustRead(t, filepath.Join(dest, "app", "routes", "api", "api.go")))
	if apiRoutes != apiAfter {
		t.Fatal("add:web must not rewrite API routes")
	}
	enabled := string(mustRead(t, filepath.Join(dest, "bootstrap", "enabled.go")))
	if !strings.Contains(enabled, `"view"`) || !strings.Contains(enabled, `"validation"`) {
		t.Fatalf("api+web enablement:\n%s", enabled)
	}
	buildAndTest(t, dest)
}

func TestFullCanonicalPresentation(t *testing.T) {
	dest := generateApp(t, "fullcanon", "--full")
	home := string(mustRead(t, filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go")))
	if !strings.Contains(home, "http.View") {
		t.Fatal("--full canonical root must be HTML")
	}
	enabled := string(mustRead(t, filepath.Join(dest, "bootstrap", "enabled.go")))
	for _, ban := range []string{`"database"`, `"auth"`, `"queue"`, `"notification"`, `"ai"`} {
		if strings.Contains(enabled, ban) {
			t.Fatalf("--full must not enable %s:\n%s", ban, enabled)
		}
	}
	meta := string(mustRead(t, filepath.Join(dest, "bootstrap", "scaffold.go")))
	if !strings.Contains(meta, `ScaffoldName      = "full"`) {
		t.Fatalf("metadata:\n%s", meta)
	}
	if _, err := os.Stat(filepath.Join(dest, "tests", "api_root_test.go")); err != nil {
		t.Fatal("canonical full includes API presentation test")
	}
	buildAndTest(t, dest)
}

func TestAddPreservesUserModifiedSource(t *testing.T) {
	dest := generateApp(t, "usermod", "--api")
	homePath := filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go")
	orig := mustRead(t, homePath)
	modified := append(append([]byte{}, orig...), []byte("\n// user edit\n")...)
	if err := os.WriteFile(homePath, modified, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := (&AddWebCommand{app: kernel.NewApplication(dest)}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, homePath)) != string(modified) {
		t.Fatal("add:web must not overwrite user-modified API home")
	}
	welcome := filepath.Join(dest, "app", "views", "welcome.html")
	webOrig := mustRead(t, welcome)
	webMod := append(append([]byte{}, webOrig...), []byte("<!-- user -->")...)
	if err := os.WriteFile(welcome, webMod, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := (&AddAPICommand{app: kernel.NewApplication(dest)}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, welcome)) != string(webMod) {
		t.Fatal("add:api must not overwrite user-modified web view")
	}
}

func TestEmptyAddWebUpgradesExactStub(t *testing.T) {
	dest := generateApp(t, "emptystub")
	home := string(mustRead(t, filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go")))
	if strings.Contains(home, "http.View") {
		t.Fatal("empty home is kernel HTML, not view")
	}
	if err := (&AddWebCommand{app: kernel.NewApplication(dest)}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	homeAfter := string(mustRead(t, filepath.Join(dest, "app", "http", "controllers", "web", "home_controller.go")))
	if !strings.Contains(homeAfter, "http.View") {
		t.Fatal("exact empty stub must upgrade to web HTML")
	}
	enabled := string(mustRead(t, filepath.Join(dest, "bootstrap", "enabled.go")))
	if !strings.Contains(enabled, `"view"`) {
		t.Fatalf("add:web must enable view:\n%s", enabled)
	}
	buildAndTest(t, dest)
}

func TestG001Preserved(t *testing.T) {
	src, err := os.ReadFile("add.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "WriteScaffoldMeta") || strings.Contains(string(src), "writeScaffoldMeta") {
		t.Fatal("add:* must not rewrite scaffold metadata")
	}
	upgrade, err := os.ReadFile(filepath.Join("..", "tests", "compatibility", "upgrade_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(upgrade)
	if strings.Contains(text, "add:web") || strings.Contains(text, "add:api") {
		t.Fatal("G-001 upgrade must not invoke add:web/add:api")
	}
}

func TestFrameworkDoesNotDependOnPackages(t *testing.T) {
	mod, err := os.ReadFile(filepath.Join("..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mod), "github.com/zatrano/packages") {
		t.Fatal("framework must not require github.com/zatrano/packages")
	}
}

func snapshotPresentation(t *testing.T, dest string) map[string][]byte {
	t.Helper()
	paths := []string{
		filepath.Join("bootstrap", "enabled.go"),
		filepath.Join("bootstrap", "addons.go"),
		filepath.Join("bootstrap", "scaffold.go"),
		filepath.Join("app", "http", "controllers", "web", "home_controller.go"),
		filepath.Join("app", "routes", "web", "web.go"),
		filepath.Join("app", "routes", "api", "api.go"),
	}
	out := map[string][]byte{}
	for _, rel := range paths {
		out[rel] = mustRead(t, filepath.Join(dest, rel))
	}
	return out
}

func assertSnapshotUnchanged(t *testing.T, dest string, snap map[string][]byte) {
	t.Helper()
	for rel, want := range snap {
		got := mustRead(t, filepath.Join(dest, rel))
		if string(got) != string(want) {
			t.Fatalf("second add:* mutated %s", rel)
		}
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func buildAndTest(t *testing.T, dest string) {
	t.Helper()
	build := exec.Command("go", "build", "./...")
	build.Dir = dest
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	testCmd := exec.Command("go", "test", "./tests")
	testCmd.Dir = dest
	if out, err := testCmd.CombinedOutput(); err != nil {
		t.Fatalf("go test ./tests: %v\n%s", err, out)
	}
}
