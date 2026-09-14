package files_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/v2/kernel/support/files"
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

func TestWriteAtomicDefaultPermAndTempDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.txt")
	if err := files.WriteAtomic(path, []byte("z"), 0); err != nil {
		t.Fatal(err)
	}
	td, err := files.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(td) })
}

func TestWriteAtomicRenameConflict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "isdir")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := files.WriteAtomic(path, []byte("x"), 0o644); err == nil {
		t.Fatal("expected rename onto directory to fail")
	}
}

func TestWriteAtomicParentIsFile(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "notdir")
	if err := os.WriteFile(base, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := files.WriteAtomic(filepath.Join(base, "child.txt"), []byte("y"), 0o644); err == nil {
		t.Fatal("expected mkdir on file to fail")
	}
	if _, err := files.TempDir(base, "p-*"); err == nil {
		t.Fatal("expected tempdir on file to fail")
	}
}
