package safepath

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	// EvalUnder's invariant is Under(resolvedRoot, resolved), not Under(lexical
	// t.TempDir(), resolved). On Windows, TEMP is often an 8.3 path
	// (C:\Users\RUNNER~1\...) while EvalSymlinks returns the long name
	// (C:\Users\runneradmin\...); Rel then looks like an escape.
	resolvedRoot, err := resolveFS(root)
	if err != nil {
		t.Fatal(err)
	}
	if !Under(resolvedRoot, got) {
		t.Fatalf("not under resolved root %s: %s", resolvedRoot, got)
	}
}

func TestResolveEmptyAndDot(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"", "."} {
		full, err := Resolve(root, p)
		if err != nil {
			t.Fatalf("%q: %v", p, err)
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			t.Fatal(err)
		}
		if full != abs {
			t.Fatalf("%q got %s want %s", p, full, abs)
		}
	}
}

func TestResolveRejectsAbsoluteAndUNC(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"C:/windows", "foo:bar"} {
		if _, err := Resolve(root, p); err == nil {
			t.Fatalf("expected reject for %q", p)
		}
	}
}

func TestEvalUnderMissingFile(t *testing.T) {
	root := t.TempDir()
	full, err := Resolve(root, "gone.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EvalUnder(root, full); err == nil {
		t.Fatal("expected Lstat error for missing path")
	}
}

func TestEvalUnderSymlinkHopLimit(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")
	if err := os.Symlink(b, a); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}
	if _, err := EvalUnder(root, a); err == nil {
		t.Fatal("expected too many symlinks")
	} else if !strings.Contains(err.Error(), "too many symlinks") && !strings.Contains(strings.ToLower(err.Error()), "too many") {
		if _, ok := err.(interface{ Error() string }); !ok {
			t.Fatalf("err=%v", err)
		}
	}
}

func TestEvalUnderInsideSymlinkAndDotSegments(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inner, "a.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(inner, link); err != nil {
		t.Skipf("symlink: %v", err)
	}
	full, err := Resolve(root, "link/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EvalUnder(root, full); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(root, "inner/./a.txt"); err != nil {
		t.Fatal(err)
	}
}

func TestStripLongPathUNCAndDrive(t *testing.T) {
	if got := stripLongPath(`\\?\C:\Windows\System32`); got != `C:\Windows\System32` {
		t.Fatalf("drive prefix: %q", got)
	}
	if got := stripLongPath(`\\?\UNC\server\share\file`); got != `\\server\share\file` {
		t.Fatalf("unc prefix: %q", got)
	}
	if got := stripLongPath(`\\?\unc\server\share`); got != `\\server\share` {
		t.Fatalf("unc lower: %q", got)
	}
	if got := stripLongPath(`/plain`); got != `/plain` {
		t.Fatalf("plain: %q", got)
	}
}

func TestResolveFSNHopLimitDirect(t *testing.T) {
	if _, err := resolveFSN(".", maxSymlinkHops+1); err == nil {
		t.Fatal("expected hop limit")
	}
}

func TestUnderAbsAndRelErrors(t *testing.T) {
	if Under("\x00", "x") {
		t.Fatal("null root")
	}
	if Under(".", "\x00") {
		t.Fatal("null candidate")
	}
	if filepath.VolumeName(`C:\`) != "" && Under(`C:\`, `D:\outside`) {
		t.Fatal("cross-volume")
	}
}

func TestSplitAbsSkipsEmptyAndDot(t *testing.T) {
	base, parts := splitAbs(filepath.Join(string(filepath.Separator), "a", ".", "b"))
	if len(parts) == 0 {
		t.Fatal("expected parts")
	}
	for _, p := range parts {
		if p == "" || p == "." {
			t.Fatalf("empty/dot part in %v base=%s", parts, base)
		}
	}
}
