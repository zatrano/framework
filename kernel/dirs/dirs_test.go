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
