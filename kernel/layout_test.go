package kernel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLayoutDirPrefersNewTree(t *testing.T) {
	dir := t.TempDir()
	app := NewApplication(dir)
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

func TestLayoutDirFallsBack(t *testing.T) {
	dir := t.TempDir()
	app := NewApplication(dir)
	if err := os.MkdirAll(filepath.Join(dir, "lang"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := LocalizationDir(app)
	want := filepath.Join(dir, "lang")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestLayoutDirForCreateUsesNewWhenMissing(t *testing.T) {
	dir := t.TempDir()
	app := NewApplication(dir)
	got := DatabaseDirForCreate(app)
	want := filepath.Join(dir, "app", "database")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
