//go:build unix

package backup_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/backup"
)

func TestBackupFilePermissions(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "app.sqlite")
	backups := filepath.Join(dir, "backups")
	if err := os.WriteFile(src, []byte("sqlite-demo"), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr := backup.New(src, backups)
	path, err := mgr.Create("perm")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(backups)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Fatalf("backup dir perm=%04o want 0700", perm)
	}
	finfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := finfo.Mode().Perm(); perm != 0o600 {
		t.Fatalf("backup file perm=%04o want 0600", perm)
	}
}
