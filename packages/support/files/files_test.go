package files_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/support/files"
)

func TestWriteAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "demo.txt")
	if err := files.WriteStringAtomic(path, "hello", 0o644); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "hello" {
		t.Fatalf("%q err=%v", raw, err)
	}
	if !files.Exists(path) {
		t.Fatal("exists")
	}
	td, err := files.TempDir(dir, "tmp-*")
	if err != nil || td == "" {
		t.Fatal(err)
	}
}
