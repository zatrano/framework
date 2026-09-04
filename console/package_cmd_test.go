package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/kernel"
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
	for _, want := range []string{"package:preset api", "WithPresetAPI()", "features", "hashid"} {
		if !strings.Contains(text, want) {
			t.Fatalf("enabled.go missing %q:\n%s", want, text)
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
	if codes["enabled.empty"] != "WARN" {
		t.Fatalf("expected enabled.empty WARN, got %#v", codes)
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
		t.Fatalf("empty EnabledAddons should not ERROR, got %d: %#v", errors, findings)
	}
}

func TestPackageDoctorFlagsLibrary(t *testing.T) {
	// Simulate a bad EnabledAddons value without rewriting repo file.
	prev := append([]string{}, bootstrap.EnabledAddons...)
	bootstrap.EnabledAddons = []string{"collection", "features"}
	defer func() { bootstrap.EnabledAddons = prev }()

	app := kernel.NewApplication(t.TempDir())
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
