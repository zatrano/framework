package tests

import (
	"context"
	"testing"

	"github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/bootstrap/addons"
)

func TestAppBootsKernelWithoutImportedPackages(t *testing.T) {
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := bootstrap.App()
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.Router() == nil {
		t.Fatal("expected router")
	}
	if app.Exceptions() == nil {
		t.Fatal("expected exceptions")
	}
	if app.Encrypter() == nil {
		t.Fatal("expected encrypter")
	}
	if app.Bound("session") {
		t.Fatal("session must not bind unless imported")
	}
	if app.Bound("db") {
		t.Fatal("database must not bind unless imported")
	}
	if app.Bound("view") {
		t.Fatal("view must not bind unless imported")
	}
	if app.Bound("auth") {
		t.Fatal("auth must not bind unless imported")
	}
	for _, m := range addons.Available() {
		if app.Bound(m.Key) {
			t.Fatalf("kernel app should not bind package %q", m.Name)
		}
	}
}

func TestWithAddonsEmptyIsKernel(t *testing.T) {
	app := bootstrap.App(bootstrap.WithAddons())
	if app == nil {
		t.Fatal("expected app")
	}
}

func TestKernelStartStopThenStartRejected(t *testing.T) {
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := bootstrap.App()
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := app.Start(); err == nil {
		t.Fatal("stopped applications must not restart")
	}
}
