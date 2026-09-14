package pkgmanager

import (
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestPackageListAndStatusHandles(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() {
		if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
			_ = c.Close()
		}
	})
	list := &PackageListCommand{app: app}
	if list.Name() != "package:list" {
		t.Fatal(list.Name())
	}
	if err := list.Handle(nil); err != nil {
		t.Fatal(err)
	}
	if err := list.Handle([]string{"--libraries"}); err != nil {
		t.Fatal(err)
	}
	if err := list.Handle([]string{"--all"}); err != nil {
		t.Fatal(err)
	}
	st := &PackageStatusCommand{app: app}
	if err := st.Handle(nil); err != nil {
		t.Fatal(err)
	}
	preset := &PackagePresetCommand{app: app}
	if err := preset.Handle(nil); err != nil {
		t.Fatal(err)
	}
	pub := &PackagePublishCommand{app: app}
	if err := pub.Handle(nil); err == nil {
		t.Fatal("publish usage")
	}
	if err := (&PackageEnableCommand{app: app}).Handle(nil); err == nil {
		t.Fatal("enable usage")
	}
	if err := (&PackageDisableCommand{app: app}).Handle(nil); err == nil {
		t.Fatal("disable usage")
	}
	if err := (&PackageInstallCommand{app: app}).Handle(nil); err == nil {
		t.Fatal("install usage")
	}
}
