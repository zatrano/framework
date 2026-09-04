package safepath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	payloads := []string{
		"../etc/passwd",
		"..\\..\\windows\\system32",
		"/../../etc/passwd",
		"../../.env",
		"foo/../../../etc/passwd",
		"..",
		"a/../../b/../../../c",
	}
	for _, p := range payloads {
		if _, err := Resolve(root, p); err == nil {
			t.Fatalf("expected reject for %q", p)
		}
	}
}

func TestResolveAllowsNested(t *testing.T) {
	root := t.TempDir()
	full, err := Resolve(root, "uploads/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !Under(root, full) {
		t.Fatalf("not under root: %s", full)
	}
	want := filepath.Join(root, "uploads", "a.txt")
	if full != want {
		t.Fatalf("got %s want %s", full, want)
	}
}

func TestResolveNullByte(t *testing.T) {
	root := t.TempDir()
	if _, err := Resolve(root, "a\x00.jpg"); err == nil {
		t.Fatal("expected null byte reject")
	}
}

func TestUnderSelf(t *testing.T) {
	root := t.TempDir()
	abs, _ := filepath.Abs(root)
	if !Under(root, abs) {
		t.Fatal("root should contain itself")
	}
	outside := filepath.Join(filepath.Dir(abs), "sibling-"+filepath.Base(abs))
	_ = os.MkdirAll(outside, 0o755)
	if Under(root, outside) {
		t.Fatal("sibling must not be under root")
	}
}
