package zipx_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/archive/zipx"
)

func TestZipCreateExtract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.zip")
	if err := zipx.Create(path, map[string][]byte{
		"hello.txt": []byte("hello"),
		"dir/a.txt": []byte("a"),
	}); err != nil {
		t.Fatal(err)
	}
	names, err := zipx.List(path)
	if err != nil || len(names) != 2 {
		t.Fatalf("%v err=%v", names, err)
	}
	out := filepath.Join(dir, "out")
	extracted, err := zipx.Extract(path, out)
	if err != nil || len(extracted) != 2 {
		t.Fatalf("%v err=%v", extracted, err)
	}
}

func TestZipSlipRejected(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "evil.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	fw, err := w.Create("../evil.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("pwn")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	out := filepath.Join(dir, "out")
	_ = os.MkdirAll(out, 0o755)
	if _, err := zipx.Extract(zipPath, out); err == nil {
		t.Fatal("expected zip-slip rejection")
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.txt")); err == nil {
		t.Fatal("evil file must not escape dest")
	}
}
