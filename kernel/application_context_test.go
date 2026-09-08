package kernel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel"
)

func TestBootstrapAndStartRemainZeroArgCompatible(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(p)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if p.starts != 1 {
		t.Fatalf("starts=%d", p.starts)
	}
	_ = app.Stop(context.Background())
}

func TestBootstrapContextNilMeansBackground(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	var ctx context.Context
	if err := app.BootstrapContext(ctx); err != nil {
		t.Fatal(err)
	}
	if !app.Bootstrapped() {
		t.Fatal("nil ctx must behave as Background")
	}
}

func TestStartContextNilMeansBackground(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(p)
	var ctx context.Context
	if err := app.StartContext(ctx); err != nil {
		t.Fatal(err)
	}
	if p.starts != 1 {
		t.Fatalf("starts=%d", p.starts)
	}
	_ = app.Stop(context.Background())
}

func TestBootstrapContextCancelBeforeRegister(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &countingProvider{}
	app.RegisterProviders(p)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := app.BootstrapContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if p.registers != 0 {
		t.Fatalf("cancelled before Register: registers=%d", p.registers)
	}
	if !app.BootstrapFailed() {
		t.Fatal("cancelled Bootstrap is a terminal BootFailed")
	}
}

func TestBootstrapContextDoesNotKillInFlightRegister(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	entered := make(chan struct{})
	release := make(chan struct{})
	second := &countingProvider{}
	app.RegisterProviders(&gateProvider{entered: entered, release: release}, second)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- app.BootstrapContext(ctx) }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("never entered Register")
	}
	cancel()
	close(release)
	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled after in-flight Register, got %v", err)
	}
	if second.registers != 0 {
		t.Fatalf("second provider must not Register after cancel, registers=%d", second.registers)
	}
}

func TestBootstrapContextCancelBeforeBoot(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &cancelAfterRegisterProvider{}
	app.RegisterProviders(p)
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	err := app.BootstrapContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if p.registers != 1 || p.boots != 0 {
		t.Fatalf("cancel checked before Boot: register=%d boot=%d", p.registers, p.boots)
	}
}

func TestStartContextCancelBeforeLifecycleStart(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(p)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := app.StartContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if p.starts != 0 {
		t.Fatalf("cancelled before Start: starts=%d", p.starts)
	}
	if !app.Bootstrapped() || app.BootstrapFailed() {
		t.Fatal("StartContext cancel must leave Booted")
	}
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	_ = app.Stop(context.Background())
}

func TestStartContextCancelBetweenProvidersCleansStarted(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	ctx, cancel := context.WithCancel(context.Background())
	first := &cancelAfterStartProvider{cancel: cancel}
	second := &lifecycleProbe{name: "second"}
	app.RegisterProviders(first, second)
	err := app.StartContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if first.starts != 1 || first.stops != 1 {
		t.Fatalf("started LP must be cleaned: start=%d stop=%d", first.starts, first.stops)
	}
	if second.starts != 0 {
		t.Fatalf("second must not Start, starts=%d", second.starts)
	}
	if !app.Bootstrapped() {
		t.Fatal("must remain Booted")
	}
}

func TestStartContextCanceledUsesBoundedCleanup(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	ok := &lifecycleProbe{name: "ok"}
	app.RegisterProviders(ok)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := app.StartContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled ctx must return ctx.Err before LP.Start, got %v", err)
	}
	if ok.starts != 0 {
		t.Fatalf("no LP.Start after cancel, starts=%d", ok.starts)
	}
	if !app.Bootstrapped() {
		t.Fatal("must remain Booted")
	}
}

func TestStartContextFailureJoinPreserved(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	startErr := errors.New("start boom")
	stopErr := errors.New("stop boom")
	ok := &lifecycleProbe{name: "ok", failStop: stopErr}
	fail := &lifecycleProbe{name: "fail", failOnce: startErr}
	app.RegisterProviders(ok, fail)
	err := app.StartContext(context.Background())
	if !errors.Is(err, startErr) || !errors.Is(err, stopErr) {
		t.Fatalf("Decision C must hold on StartContext: %v", err)
	}
	if !app.Bootstrapped() {
		t.Fatal("must remain Booted")
	}
	if err := app.StartContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	_ = app.Stop(context.Background())
}

func TestBootstrapContextCancelBetweenRegisterProviders(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	ctx, cancel := context.WithCancel(context.Background())
	first := &cancelAfterRegisterProvider{cancel: cancel}
	second := &countingProvider{}
	app.RegisterProviders(first, second)
	err := app.BootstrapContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if first.registers != 1 {
		t.Fatalf("first must Register, registers=%d", first.registers)
	}
	if second.registers != 0 {
		t.Fatalf("second must not Register after cancel, registers=%d", second.registers)
	}
	if !app.BootstrapFailed() {
		t.Fatal("cancelled Bootstrap is terminal BootFailed")
	}
}

func TestBootstrapContextCancelBetweenBootProviders(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	ctx, cancel := context.WithCancel(context.Background())
	first := &cancelAfterBootProvider{cancel: cancel}
	second := &countingProvider{}
	app.RegisterProviders(first, second)
	err := app.BootstrapContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if first.boots != 1 {
		t.Fatalf("first must Boot, boots=%d", first.boots)
	}
	if second.boots != 0 {
		t.Fatalf("second must not Boot after cancel, boots=%d", second.boots)
	}
	if !app.BootstrapFailed() {
		t.Fatal("cancelled Bootstrap is terminal BootFailed")
	}
}

func TestStopCanceledContextStillStopsProviders(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &stopSeesContext{}
	app.RegisterProviders(p)
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := app.Stop(ctx)
	if p.stops != 1 {
		t.Fatalf("Stop must still visit providers, stops=%d", p.stops)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("provider Stop should observe canceled ctx, got %v", err)
	}
}

func TestStopOnBootFailedIsNoop(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(&failRegisterProvider{}, p)
	if err := app.Bootstrap(); err == nil {
		t.Fatal("expected BootFailed")
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatalf("Stop on BootFailed must be a no-op, got %v", err)
	}
	if p.stops != 0 {
		t.Fatalf("must not Stop providers that never reached Running, stops=%d", p.stops)
	}
}

func TestStopBeforeStartIsNoop(t *testing.T) {
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
	if p.starts != 0 || p.stops != 0 {
		t.Fatalf("Booted but not Running: starts=%d stops=%d", p.starts, p.stops)
	}
}

func TestBootstrapDoesNotStartWorkers(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(p)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if p.starts != 0 {
		t.Fatalf("workers must not Start before Running, starts=%d", p.starts)
	}
}

type cancelAfterBootProvider struct {
	cancel context.CancelFunc
	boots  int
}

func (p *cancelAfterBootProvider) Register(contracts.App) error { return nil }
func (p *cancelAfterBootProvider) Boot(contracts.App) error {
	p.boots++
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

type stopSeesContext struct{ stops int }

func (p *stopSeesContext) Register(contracts.App) error { return nil }
func (p *stopSeesContext) Boot(contracts.App) error     { return nil }
func (p *stopSeesContext) Start(contracts.App) error    { return nil }
func (p *stopSeesContext) Stop(ctx context.Context) error {
	p.stops++
	return ctx.Err()
}

type cancelAfterRegisterProvider struct {
	cancel    context.CancelFunc
	registers int
	boots     int
}

func (p *cancelAfterRegisterProvider) Register(app contracts.App) error {
	p.registers++
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func (p *cancelAfterRegisterProvider) Boot(app contracts.App) error {
	p.boots++
	return nil
}

type cancelAfterStartProvider struct {
	cancel context.CancelFunc
	starts int
	stops  int
}

func (p *cancelAfterStartProvider) Register(contracts.App) error { return nil }
func (p *cancelAfterStartProvider) Boot(contracts.App) error     { return nil }
func (p *cancelAfterStartProvider) Start(contracts.App) error {
	p.starts++
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}
func (p *cancelAfterStartProvider) Stop(context.Context) error {
	p.stops++
	return nil
}
