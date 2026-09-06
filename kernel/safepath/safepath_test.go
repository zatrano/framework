package safepath

import (
	"os"
	"os/exec"
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

func TestEvalUnderRejectsOutsideJunction(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "outside-"+filepath.Base(root))
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(outside) })
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		out, jerr := exec.Command("cmd", "/c", "mklink", "/J", link, outside).CombinedOutput()
		if jerr != nil {
			t.Skipf("symlinks/junctions not supported: %v (%s)", jerr, out)
		}
		t.Cleanup(func() { _ = os.Remove(link) })
	}
	full, err := Resolve(root, "link/secret.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EvalUnder(root, full); err == nil {
		t.Fatal("expected outside symlink/junction to be rejected")
	}
}

func TestEvalUnderAllowsInside(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.css"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	full, err := Resolve(root, "app.css")
	if err != nil {
		t.Fatal(err)
	}
	got, err := EvalUnder(root, full)
	if err != nil {
		t.Fatal(err)
	}
	if !Under(root, got) {
		t.Fatalf("not under root: %s", got)
	}
}
