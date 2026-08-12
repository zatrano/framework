package filesystem_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/filesystem"
)

func TestLocalTemporaryURL(t *testing.T) {
	dir := t.TempDir()
	disk, err := filesystem.NewLocalDisk(dir)
	if err != nil {
		t.Fatal(err)
	}
	disk.SetSigningKey("secret")
	disk.SetServePath("/storage/temporary")
	if err := disk.Put("demo/hello.txt", []byte("hi")); err != nil {
		t.Fatal(err)
	}
	raw, err := disk.TemporaryURL("demo/hello.txt", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	path, ok := disk.ValidateTemporaryURL(u.RequestURI())
	if !ok || path != "demo/hello.txt" {
		t.Fatalf("validate path=%q ok=%v raw=%q", path, ok, raw)
	}
}
