package tests

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnablementDoesNotIntroduceForbiddenArchitecture(t *testing.T) {
	root := moduleRoot(t)
	bans := []string{
		"type PackageManager",
		"type LifecycleManager",
		"type UpgradeManager",
		"type DependencyManager",
		"type MigrationManager",
		"func Apply(",
		"func Rollback(",
		"zatrano.lock",
		"go mod tidy",
		"@none",
		"Meta.Migrate",
		"PackageUpgrade",
	}
	check := func(path string) {
		t.Helper()
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		src := string(body)
		for _, ban := range bans {
			if strings.Contains(src, ban) {
				t.Errorf("%s contains %s", filepath.ToSlash(strings.TrimPrefix(path, root+string(os.PathSeparator))), ban)
			}
		}
	}

	for _, rel := range []string{
		filepath.Join("console", "package_cmd.go"),
		filepath.Join("console", "package_wire.go"),
		filepath.Join("console", "package_acquire.go"),
		filepath.Join("console", "package_doctor.go"),
		filepath.Join("contracts", "app.go"),
	} {
		check(filepath.Join(root, rel))
	}

	cmdSrc, err := os.ReadFile(filepath.Join(root, "console", "package_cmd.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(cmdSrc)
	if strings.Contains(text, "/acquire") {
		t.Fatal("package_cmd.go must not import acquire")
	}
	if strings.Contains(text, "compareSemver") || strings.Contains(text, "latestCompatible") {
		t.Fatal("console must not copy registry resolution")
	}
	if !strings.Contains(text, "addons.Expand(") {
		t.Fatal("Enablement enable/disable must reuse addons.Expand")
	}

	wireSrc, err := os.ReadFile(filepath.Join(root, "console", "package_wire.go"))
	if err != nil {
		t.Fatal(err)
	}
	wire := string(wireSrc)
	if !strings.Contains(wire, "packagesModuleRequired") {
		t.Fatal("enablement must skip go get when packages is already required")
	}
	if strings.Count(wire, "github.com/zatrano/packages@v1.7.1") < 1 {
		t.Fatal("first-time enablement must pin current stable packages@v1.7.1")
	}
	if strings.Contains(wire, "packages@main") {
		t.Fatal("first-time enablement must not go get packages@main")
	}

	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mod), "github.com/zatrano/packages") {
		t.Fatal("framework must not require github.com/zatrano/packages")
	}

	err = filepath.WalkDir(filepath.Join(root, "console"), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return walkErr
		}
		base := filepath.Base(path)
		if base != "package_cmd.go" && base != "package_wire.go" && base != "package_doctor.go" && base != "package_acquire.go" {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(body)
		for _, ban := range []string{"package:uninstall", "package:upgrade", "package:downgrade"} {
			if strings.Contains(src, ban) {
				t.Errorf("%s must not add %s", base, ban)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
