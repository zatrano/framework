package filesystem

import (
	"strings"
	"testing"
)

func TestPathTraversalRejected(t *testing.T) {
	root := t.TempDir()
	disk, err := NewLocalDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	payloads := []string{
		"../.env",
		"../../etc/passwd",
		"/../../etc/passwd",
		`..\..\windows\system32\drivers\etc\hosts`,
	}
	for _, p := range payloads {
		if disk.Exists(p) {
			t.Fatalf("Exists should be false for %q", p)
		}
		if err := disk.Put(p, []byte("x")); err == nil {
			t.Fatalf("Put should reject %q", p)
		}
		if _, err := disk.Get(p); err == nil {
			t.Fatalf("Get should reject %q", p)
		}
		resolved := disk.Path(p)
		if !strings.HasPrefix(resolved, root) {
			t.Fatalf("Path(%q)=%q escaped root %q", p, resolved, root)
		}
	}
}

func TestPutGetUnderRoot(t *testing.T) {
	root := t.TempDir()
	disk, err := NewLocalDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := disk.Put("a/b.txt", []byte("ok")); err != nil {
		t.Fatal(err)
	}
	got, err := disk.Get("a/b.txt")
	if err != nil || string(got) != "ok" {
		t.Fatalf("got %q err %v", got, err)
	}
}
