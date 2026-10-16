package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPPGO001SkippedWhenGoMissing(t *testing.T) {
	root := doctorAppDir(t)
	prev := goVersionLookup
	goVersionLookup = func(string) (string, error) { return "", errGoMissing }
	t.Cleanup(func() { goVersionLookup = prev })
	if hasDoctorRule(mustDoctor(t, root), "APP-GO-001") {
		t.Fatal("missing go must skip APP-GO-001")
	}
}

func TestAPPGO001WarnsWhenToolchainIsOlder(t *testing.T) {
	root := doctorAppDir(t)
	wantDir, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	prev := goVersionLookup
	goVersionLookup = func(dir string) (string, error) {
		if dir != wantDir {
			t.Errorf("lookup dir = %q, want %q", dir, wantDir)
		}
		return "go1.25.14", nil
	}
	t.Cleanup(func() { goVersionLookup = prev })
	findings := mustDoctor(t, root)
	var got *Finding
	for i := range findings {
		if findings[i].Rule == "APP-GO-001" {
			got = &findings[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("expected APP-GO-001:\n%s", FormatDoctorText(root, findings))
	}
	if got.Severity != "warning" {
		t.Fatalf("severity %q", got.Severity)
	}
	if !strings.Contains(got.Why, "stdlib security fixes require Go >= "+RecommendedGoMinimum) {
		t.Fatalf("why %q", got.Why)
	}
}

func TestAPPGO001SilentOnRecommendedToolchain(t *testing.T) {
	root := doctorAppDir(t)
	prev := goVersionLookup
	goVersionLookup = func(string) (string, error) { return RecommendedGoMinimum, nil }
	t.Cleanup(func() { goVersionLookup = prev })
	if hasDoctorRule(mustDoctor(t, root), "APP-GO-001") {
		t.Fatal("recommended toolchain must not warn")
	}
}

func doctorAppDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}
