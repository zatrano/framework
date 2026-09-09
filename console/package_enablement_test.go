package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/bootstrap/addons"
	"github.com/zatrano/framework/v2/kernel"
)

func registerEnablementGraph(t *testing.T) {
	t.Helper()
	addons.ClearRegistry()
	t.Cleanup(addons.ClearRegistry)
	addons.Register(addons.Meta{Name: "hashing"})
	addons.Register(addons.Meta{Name: "session", Requires: []string{"hashing"}})
	addons.Register(addons.Meta{Name: "auth", Requires: []string{"session"}, Optional: []string{"redisx"}})
	addons.Register(addons.Meta{Name: "redisx"})
	addons.Register(addons.Meta{Name: "features"})
}

func TestEnableRequiresClosureDirect(t *testing.T) {
	registerEnablementGraph(t)
	app := kernel.NewApplication(t.TempDir())
	added, err := enablePackage(app, "session")
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("expected enable to add names")
	}
	names, ok := consumerManifest(app)
	if !ok {
		t.Fatal("expected manifest")
	}
	got := strings.Join(names, ",")
	if !strings.Contains(got, "session") || !strings.Contains(got, "hashing") {
		t.Fatalf("enable session must write session+hashing, got %#v", names)
	}
	if strings.Contains(got, "auth") || strings.Contains(got, "redisx") {
		t.Fatalf("must not enable unrelated or Optional, got %#v", names)
	}
}

func TestEnableRequiresClosureTransitive(t *testing.T) {
	registerEnablementGraph(t)
	app := kernel.NewApplication(t.TempDir())
	if _, err := enablePackage(app, "auth"); err != nil {
		t.Fatal(err)
	}
	names, _ := consumerManifest(app)
	want := map[string]bool{"auth": true, "session": true, "hashing": true}
	for _, n := range names {
		delete(want, n)
	}
	if len(want) != 0 {
		t.Fatalf("enable auth must write auth+session+hashing, got %#v missing %#v", names, want)
	}
	for _, n := range names {
		if n == "redisx" {
			t.Fatal("Optional redisx must not be auto-enabled")
		}
	}
}

func TestEnableOptionalNotPulled(t *testing.T) {
	registerEnablementGraph(t)
	app := kernel.NewApplication(t.TempDir())
	if _, err := enablePackage(app, "auth"); err != nil {
		t.Fatal(err)
	}
	names, _ := consumerManifest(app)
	for _, n := range names {
		if n == "redisx" {
			t.Fatal("Optional must be excluded")
		}
	}
}

func TestEnableMissingRequiresFailsBeforeMutation(t *testing.T) {
	addons.ClearRegistry()
	t.Cleanup(addons.ClearRegistry)
	addons.Register(addons.Meta{Name: "auth", Requires: []string{"missing-dep"}})
	app := kernel.NewApplication(t.TempDir())
	path := consumerEnabledPath(app)
	_, err := enablePackage(app, "auth")
	if err == nil {
		t.Fatal("expected enablement failure for missing Requires")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("must not write enabled.go on planning failure, stat=%v", statErr)
	}
}

func TestEnableWireWritesRequiresBlankImports(t *testing.T) {
	registerEnablementGraph(t)
	app := kernel.NewApplication(t.TempDir())
	if _, err := enablePackage(app, "auth"); err != nil {
		t.Fatal(err)
	}
	if err := wireEnablement(app, "auth"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(app.BasePath(), "bootstrap", "addons.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, name := range []string{"auth", "session", "hashing"} {
		if !strings.Contains(text, "github.com/zatrano/packages/"+name) {
			t.Fatalf("blank-import missing %s:\n%s", name, text)
		}
	}
	if strings.Contains(text, "packages/redisx") {
		t.Fatalf("Optional must not be wired:\n%s", text)
	}
}

func TestDisableReverseRequiresDirect(t *testing.T) {
	registerEnablementGraph(t)
	app := kernel.NewApplication(t.TempDir())
	if _, err := enablePackage(app, "auth"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(consumerEnabledPath(app))
	removed, err := disablePackage(app, "session")
	if err == nil || removed {
		t.Fatalf("disable session must fail while auth remains, removed=%v err=%v", removed, err)
	}
	if !strings.Contains(err.Error(), `"auth"`) || !strings.Contains(err.Error(), "requires") {
		t.Fatalf("error must name the dependent, got %v", err)
	}
	after, _ := os.ReadFile(consumerEnabledPath(app))
	if string(before) != string(after) {
		t.Fatal("rejected disable must not mutate enabled.go")
	}
}

func TestDisableReverseRequiresTransitive(t *testing.T) {
	registerEnablementGraph(t)
	app := kernel.NewApplication(t.TempDir())
	if _, err := enablePackage(app, "auth"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(consumerEnabledPath(app))
	removed, err := disablePackage(app, "hashing")
	if err == nil || removed {
		t.Fatalf("disable hashing must fail while session/auth remain, removed=%v err=%v", removed, err)
	}
	after, _ := os.ReadFile(consumerEnabledPath(app))
	if string(before) != string(after) {
		t.Fatal("rejected disable must not mutate enabled.go")
	}
}

func TestDisableIndependentPackage(t *testing.T) {
	registerEnablementGraph(t)
	app := kernel.NewApplication(t.TempDir())
	if _, err := enablePackage(app, "auth"); err != nil {
		t.Fatal(err)
	}
	if _, err := enablePackage(app, "features"); err != nil {
		t.Fatal(err)
	}
	removed, err := disablePackage(app, "features")
	if err != nil || !removed {
		t.Fatalf("independent disable must succeed, removed=%v err=%v", removed, err)
	}
	names, _ := consumerManifest(app)
	for _, n := range names {
		if n == "features" {
			t.Fatalf("features must be removed, got %#v", names)
		}
	}
	want := map[string]bool{"auth": true, "session": true, "hashing": true}
	for _, n := range names {
		delete(want, n)
	}
	if len(want) != 0 {
		t.Fatalf("auth closure must remain, got %#v missing %#v", names, want)
	}
}

func TestDisableAlreadyDisabledIdempotent(t *testing.T) {
	registerEnablementGraph(t)
	app := kernel.NewApplication(t.TempDir())
	if _, err := enablePackage(app, "features"); err != nil {
		t.Fatal(err)
	}
	removed, err := disablePackage(app, "features")
	if err != nil || !removed {
		t.Fatalf("first disable: removed=%v err=%v", removed, err)
	}
	before, err := os.ReadFile(consumerEnabledPath(app))
	if err != nil {
		t.Fatal(err)
	}
	removed, err = disablePackage(app, "features")
	if err != nil {
		t.Fatalf("second disable must succeed: %v", err)
	}
	if removed {
		t.Fatal("second disable must be a no-op")
	}
	after, err := os.ReadFile(consumerEnabledPath(app))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("second disable must not mutate files")
	}

	cmd := &PackageDisableCommand{app: app}
	if err := cmd.Handle([]string{"features"}); err != nil {
		t.Fatalf("package:disable already-disabled must succeed: %v", err)
	}
}

func TestDisableDoesNotCallStop(t *testing.T) {
	body, err := os.ReadFile("package_cmd.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(body)
	idx := strings.Index(src, "func (c *PackageDisableCommand) Handle")
	if idx < 0 {
		t.Fatal("missing disable handler")
	}
	end := strings.Index(src[idx:], "func (c *PackagePublishCommand)")
	if end < 0 {
		t.Fatal("could not bound disable handler")
	}
	bodyFn := src[idx : idx+end]
	for _, ban := range []string{".Stop(", "LifecycleProvider", "StartContext"} {
		if strings.Contains(bodyFn, ban) {
			t.Fatalf("package:disable must not %s", ban)
		}
	}
}

func TestDisableBlockedUsesEnablementExitCode(t *testing.T) {
	registerEnablementGraph(t)
	app := kernel.NewApplication(t.TempDir())
	if _, err := enablePackage(app, "auth"); err != nil {
		t.Fatal(err)
	}
	cmd := &PackageDisableCommand{app: app}
	err := cmd.Handle([]string{"session"})
	if err == nil {
		t.Fatal("expected disable failure")
	}
	if CodeFromError(err) != ExitEnablement {
		t.Fatalf("disable reverse-Requires must use ExitEnablement=6, got %d (%v)", CodeFromError(err), err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "package:disable") || !strings.Contains(msg, "session") || !strings.Contains(msg, "Next:") {
		t.Fatalf("disable blocker must name action, package, and next step: %v", err)
	}
}

func TestPinIntegrityPreservesExistingRequire(t *testing.T) {
	root := t.TempDir()
	mod := "module example.com/pin-app\n\ngo 1.25.0\n\nrequire github.com/zatrano/packages v2.1.0\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensurePackagesModule(root); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "github.com/zatrano/packages v2.1.0") {
		t.Fatalf("existing pin must be preserved:\n%s", text)
	}
	if strings.Contains(text, "packages@main") || strings.Contains(text, "packages v0.0.0") {
		t.Fatalf("must not rewrite pin to main:\n%s", text)
	}
}

func TestPinIntegrityReplaceIsNotARequire(t *testing.T) {
	root := t.TempDir()
	mod := "module example.com/replace-only\n\ngo 1.25.0\n\nreplace github.com/zatrano/packages => ../packages\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	if packagesModuleRequired(root) {
		t.Fatal("replace without require must not skip first-time go get")
	}
}

func TestPinIntegrityFirstTimeGetsCurrentStable(t *testing.T) {
	root := t.TempDir()
	mod := "module example.com/first-enable\n\ngo 1.25.0\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GOMODCACHE", t.TempDir())
	err := ensurePackagesModule(root)
	if err == nil {
		t.Fatal("first-time wiring must still attempt go get of the current stable tag")
	}
	if !strings.Contains(err.Error(), packagesModuleGetArg) {
		t.Fatalf("first-time error must mention %s, got %v", packagesModuleGetArg, err)
	}
	body, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "github.com/zatrano/packages") {
		t.Fatalf("failed first-time go get must not invent a packages require:\n%s", body)
	}
}

func TestEnableThenWireDoesNotClobberPin(t *testing.T) {
	registerEnablementGraph(t)
	root := t.TempDir()
	mod := "module example.com/enable-pin\n\ngo 1.25.0\n\nrequire github.com/zatrano/packages v2.1.0\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(root)
	if _, err := enablePackage(app, "auth"); err != nil {
		t.Fatal(err)
	}
	if err := wireEnablement(app, "auth"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "github.com/zatrano/packages v2.1.0") {
		t.Fatalf("enable+wire must preserve pin:\n%s", body)
	}
}

func TestGoModRequireVersionInspectsOnly(t *testing.T) {
	src := "module example.com/app\n\nrequire (\n\tgithub.com/zatrano/framework/v2 v2.0.28\n\tgithub.com/zatrano/packages v2.1.0\n)\n"
	if got := goModRequireVersion(src, "github.com/zatrano/framework/v2"); got != "v2.0.28" {
		t.Fatalf("framework pin=%q", got)
	}
	if got := goModRequireVersion(src, "github.com/zatrano/packages"); got != "v2.1.0" {
		t.Fatalf("packages pin=%q", got)
	}
	if got := goModRequireVersion(src, "github.com/example/missing"); got != "" {
		t.Fatalf("missing module must be empty, got %q", got)
	}
}

func TestPackageDoctorEnabledRequires(t *testing.T) {
	addons.ClearRegistry()
	t.Cleanup(addons.ClearRegistry)
	addons.Register(addons.Meta{Name: "session", Requires: []string{"hashing"}})
	addons.Register(addons.Meta{Name: "hashing"})
	app := kernel.NewApplication(t.TempDir())
	if err := writeEnabledAddons(app.BasePath("bootstrap", "enabled.go"), []string{"session"}); err != nil {
		t.Fatal(err)
	}
	findings := runPackageDoctor(app)
	found := false
	for _, f := range findings {
		if f.Code == "enabled.requires" && f.Level == "ERROR" && strings.Contains(f.Message, "hashing") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected enabled.requires ERROR, got %#v", findings)
	}
}

func TestPackageDoctorTransitiveRequires(t *testing.T) {
	registerEnablementGraph(t)
	app := kernel.NewApplication(t.TempDir())
	if err := writeEnabledAddons(app.BasePath("bootstrap", "enabled.go"), []string{"auth"}); err != nil {
		t.Fatal(err)
	}
	findings := runPackageDoctor(app)
	var session, hashing, redisx bool
	for _, f := range findings {
		if f.Code != "enabled.requires" || f.Level != "ERROR" {
			continue
		}
		if strings.Contains(f.Message, "session") {
			session = true
		}
		if strings.Contains(f.Message, "hashing") && strings.Contains(f.Message, "transitively") {
			hashing = true
		}
		if strings.Contains(f.Message, "redisx") {
			redisx = true
		}
	}
	if !session || !hashing {
		t.Fatalf("doctor must close A→B→C Requires, session=%v hashing=%v findings=%#v", session, hashing, findings)
	}
	if redisx {
		t.Fatal("Optional redisx must not be a mandatory enabled.requires failure")
	}
}

func TestPackageDoctorImportedDisabled(t *testing.T) {
	registerEnablementGraph(t)
	app := kernel.NewApplication(t.TempDir())
	if err := writeEnabledAddons(app.BasePath("bootstrap", "enabled.go"), []string{"hashing"}); err != nil {
		t.Fatal(err)
	}
	findings := runPackageDoctor(app)
	found := false
	for _, f := range findings {
		if f.Code == "imported.disabled" && f.Level == "WARN" && strings.Contains(f.Message, "session") {
			found = true
		}
		if f.Code == "enabled.requires" && strings.Contains(f.Message, "redisx") {
			t.Fatalf("Optional must not be treated as required: %#v", f)
		}
	}
	if !found {
		t.Fatalf("expected imported.disabled WARN for session, got %#v", findings)
	}
	hasImported := false
	for _, f := range findings {
		if f.Code == "package.imported" && strings.Contains(f.Message, "package=session") && strings.Contains(f.Message, "enabled=false") {
			hasImported = true
		}
	}
	if !hasImported {
		t.Fatalf("expected package.imported row for session, got %#v", findings)
	}
}

func TestPackageDoctorFrameworkVersion(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	findings := runPackageDoctor(app)
	found := false
	for _, f := range findings {
		if f.Code == "framework.version" {
			found = true
			if f.Level != "WARN" && f.Level != "OK" {
				t.Fatalf("framework.version level=%s", f.Level)
			}
		}
	}
	if !found {
		t.Fatalf("expected framework.version, got %#v", findings)
	}
}

func TestPackageDoctorRequiredBy(t *testing.T) {
	registerEnablementGraph(t)
	app := kernel.NewApplication(t.TempDir())
	if _, err := enablePackage(app, "auth"); err != nil {
		t.Fatal(err)
	}
	findings := runPackageDoctor(app)
	found := false
	for _, f := range findings {
		if f.Code == "enabled.required_by" && strings.Contains(f.Message, "session") && strings.Contains(f.Message, "auth") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected enabled.required_by for session←auth, got %#v", findings)
	}
}

func TestEnableUnknownReportsNextStep(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	err := (&PackageEnableCommand{app: app}).Handle([]string{"does-not-exist"})
	if err == nil {
		t.Fatal("expected enable failure")
	}
	if CodeFromError(err) != ExitEnablement {
		t.Fatalf("exit=%d want %d (%v)", CodeFromError(err), ExitEnablement, err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "package:enable") || !strings.Contains(msg, "does-not-exist") || !strings.Contains(msg, "Next:") {
		t.Fatalf("enable error must name action, package, and next step: %v", err)
	}
}
