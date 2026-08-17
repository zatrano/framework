package version_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/version"
)

func TestVersionLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "VERSION")
	if err := os.WriteFile(path, []byte("1.2.7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := version.LoadFile(path); got != "1.2.7" {
		t.Fatalf("got %q", got)
	}
	if version.Get() != "1.2.7" {
		t.Fatalf("Get=%q", version.Get())
	}
}
