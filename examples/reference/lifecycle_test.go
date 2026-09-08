package reference

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/examples/reference/internal/worker"
	"github.com/zatrano/framework/v2/kernel"
)

func TestLifecycleRegisterBootStartStop(t *testing.T) {
	app := Assemble(t.TempDir())
	t.Cleanup(func() { closeLog(t, app) })

	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if !app.Bootstrapped() || app.BootstrapFailed() {
		t.Fatal("expected Booted")
	}
	w := mustWorker(t, app)
	if w.Running() || w.Starts() != 0 {
		t.Fatal("worker must not run during Bootstrap")
	}

	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	waitRunning(t, w)
	if w.Starts() != 1 {
		t.Fatalf("starts=%d", w.Starts())
	}

	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if w.Starts() != 1 {
		t.Fatalf("Start must be once, starts=%d", w.Starts())
	}

	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if w.Running() {
		t.Fatal("worker must stop")
	}
	after := w.Ticks()
	time.Sleep(80 * time.Millisecond)
	if w.Ticks() != after {
		t.Fatal("worker ran after Stop")
	}

	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRegisterFailureIsBootFailed(t *testing.T) {
	app := Assemble(t.TempDir(), &failRegister{})
	t.Cleanup(func() { closeLog(t, app) })
	err := app.Bootstrap()
	if err == nil {
		t.Fatal("expected register failure")
	}
	if !app.BootstrapFailed() || app.Bootstrapped() {
		t.Fatal("expected BootFailed")
	}
	if !strings.Contains(err.Error(), "register") {
		t.Fatalf("err=%v", err)
	}
}

func TestBootFailureIsBootFailed(t *testing.T) {
	app := Assemble(t.TempDir(), &failBoot{})
	t.Cleanup(func() { closeLog(t, app) })
	err := app.Bootstrap()
	if err == nil {
		t.Fatal("expected boot failure")
	}
	if !app.BootstrapFailed() || app.Bootstrapped() {
		t.Fatal("expected BootFailed")
	}
	if !strings.Contains(err.Error(), "boot") {
		t.Fatalf("err=%v", err)
	}
}

func TestStartFailureLeavesBootedAndCleansUp(t *testing.T) {
	fail := &failStart{}
	app := Assemble(t.TempDir(), fail)
	t.Cleanup(func() { closeLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	w := mustWorker(t, app)
	err := app.Start()
	if err == nil {
		t.Fatal("expected start failure")
	}
	if app.BootstrapFailed() || !app.Bootstrapped() {
		t.Fatal("Start failure must leave Booted")
	}
	if w.Running() {
		t.Fatal("started worker must be cleaned up")
	}
	if fail.starts != 1 {
		t.Fatalf("fail starts=%d", fail.starts)
	}
}

func TestStopFailureKeepsFirstErrorAndContinues(t *testing.T) {
	fail := &failStop{err: errors.New("stop boom")}
	app := Assemble(t.TempDir(), fail)
	t.Cleanup(func() { closeLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	w := mustWorker(t, app)
	waitRunning(t, w)
	err := app.Stop(context.Background())
	if err == nil || !strings.Contains(err.Error(), "stop boom") {
		t.Fatalf("first Stop error must remain visible: %v", err)
	}
	if w.Running() {
		t.Fatal("shutdown must still stop the worker")
	}
	if fail.stops != 1 {
		t.Fatalf("stops=%d", fail.stops)
	}
}

func mustWorker(t *testing.T, app *kernel.Application) *worker.Ticker {
	t.Helper()
	raw, err := app.Make(keyWorker)
	if err != nil {
		t.Fatal(err)
	}
	w, ok := raw.(*worker.Ticker)
	if !ok {
		t.Fatalf("type %T", raw)
	}
	return w
}

func waitRunning(t *testing.T, w *worker.Ticker) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if w.Running() && w.Ticks() > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("worker did not start")
}

type failRegister struct{}

func (p *failRegister) Name() string                 { return "failRegister" }
func (p *failRegister) Register(contracts.App) error { return errors.New("register boom") }
func (p *failRegister) Boot(contracts.App) error     { return nil }

type failBoot struct{}

func (p *failBoot) Name() string                 { return "failBoot" }
func (p *failBoot) Register(contracts.App) error { return nil }
func (p *failBoot) Boot(contracts.App) error     { return errors.New("boot boom") }

type failStart struct {
	starts int
	stops  int
}

func (p *failStart) Name() string                 { return "failStart" }
func (p *failStart) Register(contracts.App) error { return nil }
func (p *failStart) Boot(contracts.App) error     { return nil }
func (p *failStart) Start(contracts.App) error {
	p.starts++
	return errors.New("start boom")
}
func (p *failStart) Stop(context.Context) error {
	p.stops++
	return nil
}

type failStop struct {
	err   error
	stops int
}

func (p *failStop) Name() string                 { return "failStop" }
func (p *failStop) Register(contracts.App) error { return nil }
func (p *failStop) Boot(contracts.App) error     { return nil }
func (p *failStop) Start(contracts.App) error    { return nil }
func (p *failStop) Stop(context.Context) error {
	p.stops++
	return p.err
}
