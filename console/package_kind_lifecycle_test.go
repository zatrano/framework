package console

import (
	"os"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestEnablePackageRejectsLibrary(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	_, err := enablePackage(app, "collection")
	if err == nil || !strings.Contains(err.Error(), "library package") {
		t.Fatalf("package:enable must reject libraries, err=%v", err)
	}
}

func TestEnablePackageAllowsService(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	added, err := enablePackage(app, "features")
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("service package may be enabled")
	}
}

func TestHeavyPackagesUseOwnModuleAndNoKernelLifecycleHook(t *testing.T) {
	info, ok := catalogLookup("mongo")
	if !ok || !info.Heavy {
		t.Fatal("mongo is a heavy service with its own module")
	}
	body, err := os.ReadFile("../kernel/application.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(body)
	for _, ban := range []string{"Heavy", "KindLibrary", "KindService"} {
		if strings.Contains(src, ban) {
			t.Errorf("kernel lifecycle must not special-case %s", ban)
		}
	}
}

func TestPackageInstallDoesNotAcquire(t *testing.T) {
	body, err := os.ReadFile("package_cmd.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(body)
	if !strings.Contains(src, `return "package:install"`) || !strings.Contains(src, "publishPackage(") {
		t.Fatal("package:install remains enablement plus stubs")
	}
	if strings.Contains(src, "acquire.ExecuteTargets") {
		t.Fatal("package:install must not acquire")
	}
}
