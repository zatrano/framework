package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func phase14PackagesCheckout(t *testing.T) string {
	t.Helper()
	if p := strings.TrimSpace(os.Getenv("PACKAGES_DIR")); p != "" {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	sibling := filepath.Join(filepath.Dir(moduleRoot(t)), "packages")
	if st, err := os.Stat(sibling); err == nil && st.IsDir() {
		return sibling
	}
	return ""
}

func TestPhase14FreshConsumerLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Phase 14 consumer uses go run + go build")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go executable not on PATH")
	}
	root := moduleRoot(t)
	parent := t.TempDir()
	dest := filepath.Join(parent, "freshapp")
	if strings.Contains(filepath.ToSlash(dest), filepath.ToSlash(root)+"/") || dest == root {
		t.Fatal("consumer directory must be outside the framework repository")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	newCmd := exec.CommandContext(ctx, "go", "run", "./cmd/zatrano", "new", dest, "--module", "example.com/freshapp", "--minimal", "--replace", root)
	newCmd.Dir = root
	newCmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	out, err := newCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zatrano new: %v\n%s", err, out)
	}

	for _, rel := range []string{
		"go.mod",
		filepath.Join("cmd", "app", "main.go"),
		filepath.Join("bootstrap", "enabled.go"),
		filepath.Join("bootstrap", "addons.go"),
	} {
		if _, err := os.Stat(filepath.Join(dest, rel)); err != nil {
			t.Fatalf("generated project missing %s: %v", rel, err)
		}
	}
	mod, err := os.ReadFile(filepath.Join(dest, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mod), "github.com/zatrano/framework/v2 v2.0.28") {
		t.Fatalf("generated go.mod must require v2.0.28:\n%s", mod)
	}

	build := exec.CommandContext(ctx, "go", "build", "./...")
	build.Dir = dest
	build.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	bout, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ./...: %v\n%s", err, bout)
	}

	runApp := func(args ...string) (string, error) {
		t.Helper()
		cmd := exec.CommandContext(ctx, "go", append([]string{"run", "./cmd/app"}, args...)...)
		cmd.Dir = dest
		cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	searchOut, err := runApp("package:search", "session")
	if err != nil {
		t.Fatalf("package:search: %v\n%s", err, searchOut)
	}
	if !strings.Contains(searchOut, "session") {
		t.Fatalf("search must list session:\n%s", searchOut)
	}
	if strings.Contains(searchOut, "selected:") {
		t.Fatal("package:search must not select a version")
	}

	dryOut, err := runApp("package:acquire", "session", "--dry-run")
	if err != nil {
		t.Fatalf("package:acquire --dry-run: %v\n%s", err, dryOut)
	}
	if !strings.Contains(dryOut, "dry-run") && !strings.Contains(dryOut, "go_get_arg") {
		t.Fatalf("dry-run should report the planned go get:\n%s", dryOut)
	}

	pkgDir := phase14PackagesCheckout(t)
	if pkgDir != "" {
		if !strings.Contains(string(mod), "replace github.com/zatrano/packages =>") {
			f, err := os.OpenFile(filepath.Join(dest, "go.mod"), os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.WriteString("\nreplace github.com/zatrano/packages => " + filepath.ToSlash(pkgDir) + "\n"); err != nil {
				_ = f.Close()
				t.Fatal(err)
			}
			_ = f.Close()
		}
		acqOut, err := runApp("package:acquire", "session")
		if err != nil {
			t.Fatalf("package:acquire session: %v\n%s", err, acqOut)
		}
		if strings.Contains(acqOut, "enablement: success") {
			t.Fatalf("default acquire must not enable:\n%s", acqOut)
		}
		enSession, err := runApp("package:enable", "session")
		if err != nil {
			t.Fatalf("package:enable session: %v\n%s", err, enSession)
		}
		getSession := exec.CommandContext(ctx, "go", "get", "github.com/zatrano/packages/session")
		getSession.Dir = dest
		getSession.Env = append(os.Environ(), "GOTOOLCHAIN=local")
		if gout, err := getSession.CombinedOutput(); err != nil {
			t.Fatalf("go get session after enable: %v\n%s", err, gout)
		}
		enHash, err := runApp("package:enable", "hashing")
		if err != nil {
			t.Fatalf("package:enable hashing: %v\n%s", err, enHash)
		}
		_ = enHash
		getHash := exec.CommandContext(ctx, "go", "get", "github.com/zatrano/packages/hashing")
		getHash.Dir = dest
		getHash.Env = append(os.Environ(), "GOTOOLCHAIN=local")
		if gout, err := getHash.CombinedOutput(); err != nil {
			t.Fatalf("go get hashing after enable: %v\n%s", err, gout)
		}
	} else {
		t.Log("packages checkout not found; skipping acquire/enable against github.com/zatrano/packages (set PACKAGES_DIR)")
	}

	docOut, docErr := runApp("package:doctor")
	if !strings.Contains(docOut, "framework.version") && !strings.Contains(docOut, "Summary:") {
		t.Fatalf("package:doctor must print diagnostics:\n%s", docOut)
	}
	if docErr != nil && !strings.Contains(docOut, "error(s)") {
		t.Fatalf("package:doctor: %v\n%s", docErr, docOut)
	}

	statusOut, err := runApp("package:status")
	if err != nil {
		t.Fatalf("package:status (boot): %v\n%s", err, statusOut)
	}
}
