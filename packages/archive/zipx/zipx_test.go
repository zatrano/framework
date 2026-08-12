package zipx_test

import (
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
