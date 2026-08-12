package backup_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/backup"
)

func TestBackupCreateListRestore(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "app.sqlite")
	if err := os.WriteFile(src, []byte("sqlite-demo"), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr := backup.New(src, filepath.Join(dir, "backups"))
	path, err := mgr.Create("demo")
	if err != nil {
		t.Fatal(err)
	}
	files, err := mgr.List()
	if err != nil || len(files) != 1 {
		t.Fatalf("files=%v err=%v", files, err)
	}
	_ = os.WriteFile(src, []byte("changed"), 0o644)
	if err := mgr.Restore(filepath.Base(path)); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(src)
	if string(raw) != "sqlite-demo" {
		t.Fatalf("got %q", raw)
	}
}
