package tests

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhase14DoesNotIntroduceForbiddenArchitecture(t *testing.T) {
	root := moduleRoot(t)
	bans := []string{
		"type PackageManager",
		"type LifecycleManager",
		"func Apply(",
		"func Rollback(",
		"zatrano.lock",
	}
	for _, rel := range []string{
		filepath.Join("console", "package_cmd.go"),
		filepath.Join("console", "package_wire.go"),
		filepath.Join("console", "package_acquire.go"),
		filepath.Join("console", "package_doctor.go"),
		filepath.Join("console", "package_registry.go"),
		filepath.Join("kernel", "application.go"),
		filepath.Join("contracts", "app.go"),
		filepath.Join("bootstrap", "app.go"),
	} {
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
	text := string(acq)
	if strings.Contains(text, "go mod tidy") {
		t.Fatal("package:acquire must not run go mod tidy")
	}
	if strings.Contains(text, "exec.Command") || strings.Contains(text, "os/exec") {
		t.Fatal("package:acquire must not spawn processes")
	}
	if !strings.Contains(text, `hasFlag(args, "--enable")`) {
		t.Fatal("acquisition must still gate enablement on explicit --enable")
	}

	cmdSrc, err := os.ReadFile(filepath.Join(root, "console", "package_cmd.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cmdSrc), "/acquire") {
		t.Fatal("package:install/enable must not import acquire")
	}
	if strings.Contains(string(cmdSrc), "compareSemver") || strings.Contains(string(cmdSrc), "latestCompatible") {
		t.Fatal("console must not copy registry resolution")
	}

	err = filepath.WalkDir(filepath.Join(root, "distribution", "acquire"), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return walkErr
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(body), "func Apply(") {
			t.Errorf("%s defines func Apply", filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	cliExit, err := os.ReadFile(filepath.Join(root, "console", "cli_exit.go"))
	if err != nil {
		t.Fatal(err)
	}
	exitSrc := string(cliExit)
	for _, pair := range []string{
		"ExitSuccess     = 0",
		"ExitGeneral     = 1",
		"ExitUsage       = 2",
		"ExitResolution  = 3",
		"ExitPlanning    = 4",
		"ExitAcquisition = 5",
		"ExitEnablement  = 6",
		"ExitCanceled    = 7",
	} {
		if !strings.Contains(exitSrc, pair) {
			t.Errorf("exit-code contract drifted: missing %s", pair)
		}
	}
}
