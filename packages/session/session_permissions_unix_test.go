//go:build unix

package session_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/session"
)

func TestSessionDirectoryPermissions(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = session.NewManager(dir, 120)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Fatalf("session dir perm=%04o want 0700", perm)
	}
}

func TestSessionFilePermissions(t *testing.T) {
	dir := t.TempDir()
	mgr := session.NewManager(dir, 120)
	bag, err := mgr.Start("")
	if err != nil {
		t.Fatal(err)
	}
	bag.Put("k", "v")
	if err := mgr.Save(bag); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, bag.ID()))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("session file perm=%04o want 0600", perm)
	}
}
