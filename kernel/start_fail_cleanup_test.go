package kernel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestStartFailureCleanupSucceedsReturnsStartError(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	startErr := errors.New("start boom")
	ok := &lifecycleProbe{name: "ok"}
	fail := &lifecycleProbe{name: "fail", failOnce: startErr}
	app.RegisterProviders(ok, fail)
	err := app.Start()
	if err == nil {
		t.Fatal("expected Start error")
	}
	if !errors.Is(err, startErr) {
		t.Fatalf("Start error must remain identifiable, got %v", err)
	}
	if !app.Bootstrapped() || app.BootstrapFailed() {
		t.Fatal("state must remain Booted")
	}
	if ok.stops != 1 || fail.stops != 0 {
		t.Fatalf("cleanup started-only: ok.stops=%d fail.stops=%d", ok.stops, fail.stops)
	}
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	_ = app.Stop(context.Background())
}

func TestStartFailureCleanupFailureIsJoined(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	startErr := errors.New("start boom")
	stopErr := errors.New("stop boom")
	ok := &lifecycleProbe{name: "ok", failStop: stopErr}
	fail := &lifecycleProbe{name: "fail", failOnce: startErr}
	app.RegisterProviders(ok, fail)
	err := app.Start()
	if err == nil {
		t.Fatal("expected Start error")
	}
	if !errors.Is(err, startErr) {
		t.Fatalf("Start error must remain identifiable, got %v", err)
	}
	if !errors.Is(err, stopErr) {
		t.Fatalf("cleanup error must remain inspectable, got %v", err)
	}
	if !app.Bootstrapped() || app.BootstrapFailed() {
		t.Fatal("state must remain Booted")
	}
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	_ = app.Stop(context.Background())
}

func TestStartFailureMultipleCleanupErrorsAreJoined(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	startErr := errors.New("start boom")
	stopA := errors.New("stop a")
	stopB := errors.New("stop b")
	a := &lifecycleProbe{name: "a", failStop: stopA}
	b := &lifecycleProbe{name: "b", failStop: stopB}
	fail := &lifecycleProbe{name: "fail", failOnce: startErr}
	app.RegisterProviders(a, b, fail)
	err := app.Start()
	if err == nil {
		t.Fatal("expected Start error")
	}
	if !errors.Is(err, startErr) {
		t.Fatalf("Start error missing: %v", err)
	}
	if !errors.Is(err, stopA) || !errors.Is(err, stopB) {
		t.Fatalf("both cleanup errors must be inspectable: %v", err)
	}
	if a.stops != 1 || b.stops != 1 || fail.stops != 0 {
		t.Fatalf("stops a=%d b=%d fail=%d", a.stops, b.stops, fail.stops)
	}
	if !app.Bootstrapped() {
		t.Fatal("state must remain Booted")
	}
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	_ = app.Stop(context.Background())
}
