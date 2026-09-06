package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/bootstrap"
	"github.com/zatrano/framework/v2/kernel"
)

func TestParseEnabledAddons(t *testing.T) {
	src := `package bootstrap

var EnabledAddons = []string{
	"mongo",
	"oauth",
}
`
	got := parseEnabledAddons(src)
	if len(got) != 2 || got[0] != "mongo" || got[1] != "oauth" {
		t.Fatalf("got %#v", got)
	}
	if strings.Contains(strings.Join(got, ","), " ") {
		t.Fatal("unexpected spaces")
	}
}

func TestPresetNamesCoverAPIWeb(t *testing.T) {
	names := map[string]bool{}
	for _, n := range []string{"api", "web"} {
		names[n] = false
	}
	for _, n := range bootstrap.PresetNames() {
		if _, ok := names[n]; ok {
			names[n] = true
		}
	}
	for n, ok := range names {
		if !ok {
			t.Fatalf("missing preset %q", n)
		}
	}
}

func TestWriteEnabledAddonsMentionsPresets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "enabled.go")
	if err := writeEnabledAddons(path, []string{"features", "hashid"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{"package:preset api", "WithPresetAPI()", "features", "hashid", "RegisterEnablement", "fwbootstrap"} {
		if !strings.Contains(text, want) {
			t.Fatalf("enabled.go missing %q:\n%s", want, text)
		}
	}
	for _, stale := range []string{"Kernel()", "Minimal()", "MinimalApp"} {
		if strings.Contains(text, stale) {
			t.Fatalf("enabled.go still mentions removed boot API %q:\n%s", stale, text)
		}
	}
}

func TestPublishPackagesQuietOnlyStubbed(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	published, skipped, err := publishPackagesQuiet(app, []string{"features", "oauth", "hashid"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if published != 1 {
		t.Fatalf("expected 1 publish (oauth), got published=%d skipped=%d", published, skipped)
	}
	if _, err := os.Stat(filepath.Join(app.BasePath("config"), "oauth.go")); err != nil {
		t.Fatal(err)
	}
	// second run skips existing
	published, skipped, err = publishPackagesQuiet(app, []string{"oauth"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if published != 0 || skipped != 1 {
		t.Fatalf("expected skip, got published=%d skipped=%d", published, skipped)
	}
}

func TestPackageDoctorEmptyEnabled(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	findings := runPackageDoctor(app)
	codes := map[string]string{}
	for _, f := range findings {
		codes[f.Code] = f.Level
	}
	if codes["enabled.nomanifest"] != "WARN" {
		t.Fatalf("expected enabled.nomanifest WARN, got %#v", codes)
	}
	if codes["catalog.providers"] != "OK" {
		t.Fatalf("expected catalog.providers OK, got %#v", codes)
	}
	errors := 0
	for _, f := range findings {
		if f.Level == "ERROR" {
			errors++
		}
	}
	if errors != 0 {
		t.Fatalf("empty enablement manifest should not ERROR, got %d: %#v", errors, findings)
	}
}

func TestPackageDoctorFlagsLibrary(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	if err := writeEnabledAddons(app.BasePath("bootstrap", "enabled.go"), []string{"collection", "features"}); err != nil {
		t.Fatal(err)
	}

	findings := runPackageDoctor(app)
	found := false
	for _, f := range findings {
		if f.Code == "enabled.library" && f.Level == "ERROR" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected enabled.library ERROR, got %#v", findings)
	}
}

func TestPackageDoctorIgnoresFrameworkEnabledAddons(t *testing.T) {
	prev := append([]string{}, bootstrap.EnabledAddons...)
	bootstrap.EnabledAddons = []string{"collection", "features"}
	defer func() { bootstrap.EnabledAddons = prev }()

	app := kernel.NewApplication(t.TempDir())
	findings := runPackageDoctor(app)
	for _, f := range findings {
		if f.Code == "enabled.library" {
			t.Fatalf("doctor must not read framework bootstrap.EnabledAddons, got %#v", findings)
		}
	}
	codes := map[string]string{}
	for _, f := range findings {
		codes[f.Code] = f.Level
	}
	if codes["enabled.nomanifest"] != "WARN" {
		t.Fatalf("expected enabled.nomanifest WARN, got %#v", codes)
	}
}

func TestEnableDisableReadsConsumerManifest(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	added, err := enablePackage(app, "features")
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("expected features to be added")
	}
	names, ok := consumerManifest(app)
	if !ok || len(names) != 1 || names[0] != "features" {
		t.Fatalf("manifest after enable: ok=%v names=%#v", ok, names)
	}
	added, err = enablePackage(app, "features")
	if err != nil {
		t.Fatal(err)
	}
	if added {
		t.Fatal("second enable should be a no-op")
	}
	removed, err := disablePackage(app, "features")
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Fatal("expected disable to remove features")
	}
	names, ok = consumerManifest(app)
	if !ok || len(names) != 0 {
		t.Fatalf("manifest after disable: ok=%v names=%#v", ok, names)
	}
	body, err := os.ReadFile(consumerEnabledPath(app))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "RegisterEnablement") {
		t.Fatalf("disable rewrite lost init registration:\n%s", body)
	}
}

func TestPackageDoctorEmptyManifestFile(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	if err := writeEnabledAddons(app.BasePath("bootstrap", "enabled.go"), nil); err != nil {
		t.Fatal(err)
	}
	findings := runPackageDoctor(app)
	codes := map[string]string{}
	for _, f := range findings {
		codes[f.Code] = f.Level
	}
	if codes["enabled.empty"] != "WARN" {
		t.Fatalf("expected enabled.empty WARN, got %#v", codes)
	}
	if codes["enabled.nomanifest"] != "" {
		t.Fatalf("empty file is a registered manifest, not missing: %#v", codes)
	}
}
