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

func TestFreshApplicationErgonomics(t *testing.T) {
	if testing.Short() {
		t.Skip("Application ergonomics consumer uses go run + go build")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go executable not on PATH")
	}
	root := moduleRoot(t)
	parent := t.TempDir()
	dest := filepath.Join(parent, "freshapp")
	if dest == root || strings.Contains(filepath.ToSlash(dest), filepath.ToSlash(root)+"/") {
		t.Fatal("consumer directory must be outside the framework repository")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	newCmd := exec.CommandContext(ctx, "go", "run", "./cmd/zatrano", "new", dest, "--module", "example.com/freshapp", "--replace", root)
	newCmd.Dir = root
	newCmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	out, err := newCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zatrano new: %v\n%s", err, out)
	}

	mod, err := os.ReadFile(filepath.Join(dest, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(mod)
	if !strings.Contains(text, "module example.com/freshapp") {
		t.Fatalf("go.mod module:\n%s", text)
	}
	if !strings.Contains(text, "github.com/zatrano/framework/v2 v2.3.1") {
		t.Fatalf("go.mod must require v2.3.1:\n%s", text)
	}
	if strings.Contains(text, "v2-dev") {
		t.Fatalf("go.mod must not use v2-dev:\n%s", text)
	}
	if !strings.Contains(text, "github.com/zatrano/packages") {
		t.Fatalf("generated go.mod must pin packages:\n%s", text)
	}

	build := exec.CommandContext(ctx, "go", "build", "./...")
	build.Dir = dest
	build.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	if bout, err := build.CombinedOutput(); err != nil {
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

	verOut, err := runApp("--version")
	if err != nil {
		t.Fatalf("--version: %v\n%s", err, verOut)
	}
	if !strings.Contains(verOut, "2.3.1") {
		t.Fatalf("version must report 2.3.1:\n%s", verOut)
	}

	helpOut, err := runApp("--help")
	if err != nil {
		t.Fatalf("--help: %v\n%s", err, helpOut)
	}
	if !strings.Contains(helpOut, "serve") || !strings.Contains(helpOut, "package:doctor") {
		t.Fatalf("help must list commands:\n%s", helpOut)
	}

	aboutOut, err := runApp("about")
	if err != nil {
		t.Fatalf("about (boot): %v\n%s", err, aboutOut)
	}

	testCmd := exec.CommandContext(ctx, "go", "test", "./tests", "-count=1", "-run", "TestLifecycleStartStopWithoutInfrastructure")
	testCmd.Dir = dest
	testCmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	if tout, err := testCmd.CombinedOutput(); err != nil {
		t.Fatalf("generated Start/Stop test: %v\n%s", err, tout)
	}
}

func TestErgonomicsDoesNotIntroduceForbiddenArchitecture(t *testing.T) {
	root := moduleRoot(t)
	bans := []string{
		"type PackageManager",
		"type LifecycleManager",
		"type ApplicationManager",
		"type ConfigManager",
		"type RuntimeManager",
		"type ServiceLocator",
		"type DependencyContainer",
		"func Apply(",
		"func Rollback(",
		"zatrano.lock",
	}
	files := []string{
		filepath.Join("console", "console.go"),
		filepath.Join("console", "new.go"),
		filepath.Join("kernel", "application.go"),
		filepath.Join("kernel", "env", "env.go"),
		filepath.Join("contracts", "app.go"),
		filepath.Join("bootstrap", "app.go"),
	}
	for _, rel := range files {
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		src := string(body)
		for _, ban := range bans {
			if strings.Contains(src, ban) {
				t.Errorf("%s contains %s", filepath.ToSlash(rel), ban)
			}
		}
	}

	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mod), "github.com/zatrano/packages") {
		t.Fatal("framework must not require github.com/zatrano/packages")
	}

	kernelSrc, err := os.ReadFile(filepath.Join(root, "kernel", "application.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(kernelSrc), "MeetsFrameworkMin") {
		t.Fatal("Bootstrap/Start must not enforce framework_min")
	}

	acq, err := os.ReadFile(filepath.Join(root, "console", "package_acquire.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(acq), "go mod tidy") {
		t.Fatal("package:acquire must not run go mod tidy")
	}

	cli, err := os.ReadFile(filepath.Join(root, "console", "cli_exit.go"))
	if err != nil {
		t.Fatal(err)
	}
	exitSrc := string(cli)
	for _, pair := range []string{
		"ExitUsage       = 2",
		"ExitResolution  = 3",
		"ExitAcquisition = 5",
		"ExitEnablement  = 6",
		"ExitCanceled    = 7",
	} {
		if !strings.Contains(exitSrc, pair) {
			t.Errorf("exit-code contract drifted: missing %s", pair)
		}
	}
}
