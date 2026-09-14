package console

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestRootCLICommandHandles(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	t.Cleanup(func() {
		if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
			_ = c.Close()
		}
	})
	if err := (&InspireCommand{}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	if err := (&AboutCommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	if err := (&VersionCommand{}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	if err := (&EnvEncryptCommand{app: app}).Handle(nil); err == nil {
		t.Fatal("encrypt usage")
	}
	if err := (&EnvDecryptCommand{app: app}).Handle(nil); err == nil {
		t.Fatal("decrypt usage")
	}
	if err := (&MakeServiceCommand{app: app}).Handle(nil); err == nil {
		t.Fatal("service name")
	}
	if err := (&MakeServiceCommand{app: app}).Handle([]string{"Order"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "app", "services", "order_service.go")); err != nil {
		t.Fatal(err)
	}
	if err := (&MakeExceptionCommand{app: app}).Handle(nil); err == nil {
		t.Fatal("exception name")
	}
	if err := (&MakeExceptionCommand{app: app}).Handle([]string{"Gone"}); err != nil {
		t.Fatal(err)
	}
	if err := (&MakeTestCommand{app: app}).Handle(nil); err == nil {
		t.Fatal("test name")
	}
	if err := (&MakeTestCommand{app: app}).Handle([]string{"Health"}); err != nil {
		t.Fatal(err)
	}
	if err := (&StorageLinkCommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	if err := (&StorageLinkCommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	if err := (&ConfigClearCommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	if err := (&RouteClearCommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	if err := (&RouteListCommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	if err := (&ConfigCacheCommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	if err := (&RouteCacheCommand{app: app}).Handle(nil); err != nil {
		t.Fatal(err)
	}
	if err := (&DeployCheckCommand{app: app}).Handle(nil); err == nil {
		t.Fatal("deploy:check should report missing docker files")
	}
	cli := New(app)
	if _, ok := cli.Commands()["inspire"]; !ok {
		t.Fatal("inspire registered")
	}
	if err := (&MakeControllerCommand{app: app}).Handle(nil); err == nil {
		t.Fatal("controller name")
	}
	if err := (&MakeMiddlewareCommand{app: app}).Handle(nil); err == nil {
		t.Fatal("middleware name")
	}
	if err := (&MakeMiddlewareCommand{app: app}).Handle([]string{"Auth"}); err != nil {
		t.Fatal(err)
	}
	tinker := &TinkerCommand{app: app}
	for _, args := range [][]string{
		{"help"}, {"app"}, {"config"}, {"config", "app.name"},
		{"env"}, {"env", "PATH"}, {"metrics"}, {"nope"}, {"exit"},
	} {
		if err := tinker.Handle(args); err != nil {
			t.Fatalf("tinker %v: %v", args, err)
		}
	}
	n, err := mergePackageEnvFile(filepath.Join(dir, ".env"), "session", "SESSION_DRIVER=file\n")
	if err != nil || n != 1 {
		t.Fatalf("merge env n=%d err=%v", n, err)
	}
	n, err = mergePackageEnvFile(filepath.Join(dir, ".env"), "session", "SESSION_DRIVER=file\n")
	if err != nil || n != 0 {
		t.Fatalf("idempotent merge n=%d err=%v", n, err)
	}
}
