package filesystem_test

import (
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/filesystem"
)

func TestDirectoryHelpers(t *testing.T) {
	root := t.TempDir()
	disk, err := filesystem.NewLocalDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	_ = disk.MakeDirectory("docs/a")
	_ = disk.MakeDirectory("docs/b")
	_ = disk.Put("docs/readme.txt", []byte("hi"))
	_ = disk.Put("docs/a/one.txt", []byte("1"))
	_ = disk.Put("docs/b/two.txt", []byte("2"))

	dirs, err := disk.Directories("docs")
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 2 {
		t.Fatalf("dirs=%v", dirs)
	}
	files, err := disk.Files("docs")
	if err != nil || len(files) != 1 {
		t.Fatalf("files=%v err=%v", files, err)
	}
	all, err := disk.AllFiles("docs")
	if err != nil || len(all) != 3 {
		t.Fatalf("all=%v err=%v", all, err)
	}

	mgr := filesystem.NewManager("local", map[string]filesystem.Disk{"local": disk})
	listed, err := mgr.AllFiles("docs")
	if err != nil || len(listed) != 3 {
		t.Fatalf("manager all=%v err=%v", listed, err)
	}
	if err := mgr.DeleteDirectory("docs/a"); err != nil {
		t.Fatal(err)
	}
	if disk.Exists("docs/a/one.txt") {
		t.Fatal("expected docs/a deleted")
	}
	if !disk.Exists(filepath.ToSlash("docs/readme.txt")) && !disk.Exists("docs/readme.txt") {
		t.Fatal("readme should remain")
	}
}
