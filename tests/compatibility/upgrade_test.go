package compatibility_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// nestedPackagesModules must match console.nestedPackagesModules and starter-smoke.sh.
var nestedPackagesModules = []string{
	"database/driver/sqlite",
	"database/driver/mysql",
	"database/driver/pgsql",
	"database/driver/mssql",
	"database/driver/oracle",
	"database/driver/mongo",
	"mongo",
	"webauthn",
	"qr",
}

// TestGeneratedAppUpgradesAcrossFrameworkRevisions is the G-001 measurement
// harness: generate with framework revision A, consume B, application source
// (except go.mod/go.sum) must not change, and go test ./... must pass.
//
// dirty:  A=HEAD, B=worktree
// clean:  A=HEAD~1, B=HEAD
// A==B:   skip
func TestGeneratedAppUpgradesAcrossFrameworkRevisions(t *testing.T) {
	if testing.Short() {
		t.Skip("G-001 worktree + generated go test")
	}
	bRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(bRoot, ".git")); err != nil {
		t.Skip("G-001 requires a git checkout")
	}
	packages := packagesDir(bRoot)
	if packages == "" {
		t.Skip("G-001 requires a packages checkout (sibling packages/ or PACKAGES_DIR)")
	}

	dirty := gitPorcelain(t, bRoot) != ""
	head := gitOut(t, bRoot, "rev-parse", "HEAD")
	var aRev string
	if dirty {
		aRev = head
	} else {
		out, err := git(bRoot, "rev-parse", "HEAD~1")
		if err != nil {
			t.Skipf("G-001 needs HEAD~1 (fetch-depth >= 2): %v", err)
		}
		aRev = strings.TrimSpace(string(out))
	}

	if !dirty {
		if err := runGit(bRoot, "diff", "--quiet", "HEAD~1", "HEAD"); err == nil {
			t.Skip("G-001: A == B (no framework delta)")
		}
	}

	t.Logf("G-001 dirty=%v A=%s B=%s", dirty, aRev, bRoot)

	aDir := filepath.Join(filepath.Dir(bRoot), "g001-fw-A")
	_ = os.RemoveAll(aDir)
	if err := runGit(bRoot, "worktree", "add", "--detach", aDir, aRev); err != nil {
		t.Fatalf("worktree A: %v", err)
	}
	t.Cleanup(func() {
		_ = runGit(bRoot, "worktree", "remove", "--force", aDir)
		_ = os.RemoveAll(aDir)
	})

	appDir := filepath.Join(t.TempDir(), "app")
	gen := exec.Command("go", "run", "./cmd/zatrano", "new", appDir, "--module", "example.com/g001", "--replace", aDir)
	gen.Dir = aDir
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("generate @ A: %v\n%s", err, out)
	}

	before := snapshotAppSource(t, appDir)
	pinModuleReplaces(t, appDir, bRoot, packages)
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = appDir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, out)
	}

	after := snapshotAppSource(t, appDir)
	if delta := snapshotDelta(before, after); delta != "" {
		t.Fatalf("application source changed after consume(B):\n%s", delta)
	}
	assertConsumesFramework(t, appDir, bRoot)

	test := exec.Command("go", "test", "./...")
	test.Dir = appDir
	if out, err := test.CombinedOutput(); err != nil {
		t.Fatalf("go test ./... against B: %v\n%s", err, out)
	}
}

func gitPorcelain(t *testing.T, root string) string {
	t.Helper()
	return gitOut(t, root, "status", "--porcelain")
}

func gitOut(t *testing.T, root string, args ...string) string {
	t.Helper()
	out, err := git(root, args...)
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func git(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	return cmd.CombinedOutput()
}

func runGit(root string, args ...string) error {
	out, err := git(root, args...)
	if err != nil {
		return fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return nil
}

func packagesDir(frameworkRoot string) string {
	if p := strings.TrimSpace(os.Getenv("PACKAGES_DIR")); p != "" {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	candidate := filepath.Join(filepath.Dir(frameworkRoot), "packages")
	if st, err := os.Stat(candidate); err == nil && st.IsDir() {
		abs, _ := filepath.Abs(candidate)
		return abs
	}
	return ""
}

func pinModuleReplaces(t *testing.T, appDir, framework, packages string) {
	t.Helper()
	modEdit(t, appDir, "-replace", "github.com/zatrano/framework="+framework)
	modEdit(t, appDir, "-replace", "github.com/zatrano/packages="+packages)
	for _, rel := range nestedPackagesModules {
		p := filepath.Join(packages, filepath.FromSlash(rel))
		if _, err := os.Stat(filepath.Join(p, "go.mod")); err != nil {
			continue
		}
		modEdit(t, appDir, "-replace", "github.com/zatrano/packages/"+rel+"="+p)
	}
}

func modEdit(t *testing.T, appDir string, args ...string) {
	t.Helper()
	cmd := exec.Command("go", append([]string{"mod", "edit"}, args...)...)
	cmd.Dir = appDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod edit %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func assertConsumesFramework(t *testing.T, appDir, framework string) {
	t.Helper()
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/zatrano/framework")
	cmd.Dir = appDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list framework: %v\n%s", err, out)
	}
	got := strings.TrimSpace(string(out))
	if !sameDir(got, framework) {
		t.Fatalf("generated app did not consume B: go list Dir=%q want %q", got, framework)
	}
}

func sameDir(a, b string) bool {
	sa, err1 := os.Stat(a)
	sb, err2 := os.Stat(b)
	if err1 != nil || err2 != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return os.SameFile(sa, sb)
}

func snapshotAppSource(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		base := filepath.Base(path)
		if base == "go.mod" || base == "go.sum" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		out[rel] = hex.EncodeToString(h.Sum(nil))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("empty application source snapshot")
	}
	return out
}

func snapshotDelta(before, after map[string]string) string {
	var b strings.Builder
	for path, sum := range before {
		got, ok := after[path]
		if !ok {
			fmt.Fprintf(&b, "  removed %s\n", path)
			continue
		}
		if got != sum {
			fmt.Fprintf(&b, "  changed %s\n", path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			fmt.Fprintf(&b, "  added %s\n", path)
		}
	}
	return b.String()
}
