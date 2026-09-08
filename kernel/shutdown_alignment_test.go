package kernel_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel"
)

// Phase 11 Contract D: normal Stop vs Start-failure cleanup.
//
// Start-failure cleanup stops only LifecycleProviders that returned nil
// from Start (the failed provider is not Stop'd). Normal Stop, once
// Running, iterates every LifecycleProvider on the provider slice in
// reverse registration order.

type neverStartedLP struct {
	stops int
}

func (p *neverStartedLP) Register(contracts.App) error { return nil }
func (p *neverStartedLP) Boot(contracts.App) error     { return nil }
func (p *neverStartedLP) Start(contracts.App) error {
	return errors.New("never running")
}
func (p *neverStartedLP) Stop(context.Context) error {
	p.stops++
	return nil
}

func TestStopWhileRunningReverseProviderOrder(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	var order []string
	a := &lifecycleProbe{name: "a", order: &order}
	b := &lifecycleProbe{name: "b", order: &order}
	c := &lifecycleProbe{name: "c", order: &order}
	app.RegisterProviders(a, b, c)
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []string{"start:a", "start:b", "start:c", "stop:c", "stop:b", "stop:a"}
	if len(order) != len(want) {
		t.Fatalf("order=%v", order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order=%v want %v", order, want)
		}
	}
}

func TestStopWhileBootedIsNoop(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(p)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p.stops != 0 {
		t.Fatalf("Stop while Booted must be no-op, stops=%d", p.stops)
	}
}

func TestStopWhileStoppedIsNoop(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(p)
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p.stops != 1 {
		t.Fatalf("second Stop must be no-op, stops=%d", p.stops)
	}
}

func TestConcurrentStopInvokesProvidersOnce(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(p)
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = app.Stop(context.Background())
		}()
	}
	wg.Wait()
	if p.stops != 1 {
		t.Fatalf("concurrent Stop called worker %d times", p.stops)
	}
}

func TestStartFailureCleanupStopsOnlySuccessfullyStarted(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	ok := &lifecycleProbe{name: "ok"}
	never := &neverStartedLP{}
	app.RegisterProviders(ok, never)
	if err := app.Start(); err == nil {
		t.Fatal("expected Start failure")
	}
	if ok.stops != 1 {
		t.Fatalf("successful Start must be cleaned up, stops=%d", ok.stops)
	}
	if never.stops != 0 {
		t.Fatalf("failed Start must not be Stop'd, stops=%d", never.stops)
	}
}

func TestNormalStopUsesProviderSliceLifecycleProviders(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	// A provider that is a LifecycleProvider after a successful Start is
	// included in the slice Stop, even if Start was a no-op success.
	a := &lifecycleProbe{name: "a"}
	b := &lifecycleProbe{name: "b"}
	app.RegisterProviders(a, b)
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if a.stops != 1 || b.stops != 1 {
		t.Fatalf("normal Stop must visit every LP on the slice, a=%d b=%d", a.stops, b.stops)
	}
}

func TestStopReturnsFirstStopError(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	first := errors.New("first stop")
	second := errors.New("second stop")
	a := &lifecycleProbe{name: "a", failStop: second}
	b := &lifecycleProbe{name: "b", failStop: first}
	app.RegisterProviders(a, b)
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	err := app.Stop(context.Background())
	if !errors.Is(err, first) {
		t.Fatalf("normal Stop keeps the first reverse-order error, got %v", err)
	}
	if errors.Is(err, second) {
		t.Fatal("normal Stop must not join later Stop errors")
	}
}
