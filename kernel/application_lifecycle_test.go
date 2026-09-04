package kernel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/kernel/config"
	"github.com/zatrano/framework/kernel/container"
	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

type countingProvider struct {
	registers int
	boots     int
}

func (p *countingProvider) Register(app contracts.App) error {
	p.registers++
	app.Router().Get("/once", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	return nil
}

func (p *countingProvider) Boot(app contracts.App) error {
	p.boots++
	return nil
}

func closeAppLog(t *testing.T, app *kernel.Application) {
	t.Helper()
	if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
		_ = c.Close()
	}
}

func TestBootstrapIsIdempotent(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &countingProvider{}
	app.RegisterProviders(p)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if p.registers != 1 || p.boots != 1 {
		t.Fatalf("register=%d boot=%d", p.registers, p.boots)
	}
	if !app.Bootstrapped() {
		t.Fatal("expected bootstrapped")
	}
	r := routing.From(app)
	if r == nil || !r.Frozen() {
		t.Fatal("expected frozen router")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected freeze panic")
		}
	}()
	r.Get("/late", func(req *http.Request) *http.Response {
		return http.Text("no")
	})
}

func TestRegisterProvidersAfterBootstrapPanics(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	app.RegisterProviders(&countingProvider{})
}

func TestConfigFrozenAfterBootstrap(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	repo, ok := app.Config().(*config.Repository)
	if !ok || repo == nil || !repo.Frozen() {
		t.Fatal("expected frozen config repository")
	}
}

type failRegisterProvider struct{}

func (p *failRegisterProvider) Register(app contracts.App) error {
	return errors.New("boom")
}

func (p *failRegisterProvider) Boot(app contracts.App) error { return nil }

func TestBootstrapFailureIsTerminal(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &countingProvider{}
	app.RegisterProviders(p, &failRegisterProvider{})
	if err := app.Bootstrap(); err == nil {
		t.Fatal("expected bootstrap error")
	}
	if !app.BootstrapFailed() {
		t.Fatal("expected BootstrapFailed")
	}
	if err := app.Bootstrap(); err == nil {
		t.Fatal("retry should be rejected")
	}
	if p.registers != 1 || p.boots != 0 {
		t.Fatalf("register=%d boot=%d (retry would duplicate side effects)", p.registers, p.boots)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	app.RegisterProviders(&countingProvider{})
}

func TestContainerFrozenAfterBootstrap(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	c, ok := app.Container().(*container.Container)
	if !ok || c == nil || !c.Frozen() {
		t.Fatal("expected frozen container")
	}
	if _, err := c.Make("app"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected bind panic")
		}
	}()
	c.Singleton("late", func() any { return 1 })
}

type lifecycleProbe struct {
	starts int
	stops  int
	order  *[]string
	name   string
	fail   error
}

func (p *lifecycleProbe) Register(app contracts.App) error { return nil }
func (p *lifecycleProbe) Boot(app contracts.App) error     { return nil }
func (p *lifecycleProbe) Start(app contracts.App) error {
	if p.order != nil {
		*p.order = append(*p.order, "start:"+p.name)
	}
	p.starts++
	return p.fail
}
func (p *lifecycleProbe) Stop(ctx context.Context) error {
	if p.order != nil {
		*p.order = append(*p.order, "stop:"+p.name)
	}
	p.stops++
	return nil
}

func TestLifecycleStartStopOrder(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	var order []string
	a := &lifecycleProbe{name: "a", order: &order}
	b := &lifecycleProbe{name: "b", order: &order}
	app.RegisterProviders(a, b)
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if a.starts != 1 || b.starts != 1 {
		t.Fatalf("starts a=%d b=%d", a.starts, b.starts)
	}
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if a.starts != 1 {
		t.Fatal("Start should be idempotent while running")
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if a.stops != 1 || b.stops != 1 {
		t.Fatalf("stops a=%d b=%d", a.stops, b.stops)
	}
	want := []string{"start:a", "start:b", "stop:b", "stop:a"}
	if len(order) != len(want) {
		t.Fatalf("order=%v", order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order=%v", order)
		}
	}
	if err := app.Start(); err == nil {
		t.Fatal("expected start-after-stop error")
	}
}

func TestLifecycleStartFailureStopsPrior(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	ok := &lifecycleProbe{name: "ok"}
	fail := &lifecycleProbe{name: "fail", fail: errors.New("nope")}
	app.RegisterProviders(ok, fail)
	if err := app.Start(); err == nil {
		t.Fatal("expected start error")
	}
	if ok.starts != 1 || ok.stops != 1 {
		t.Fatalf("ok start=%d stop=%d", ok.starts, ok.stops)
	}
	if fail.starts != 1 || fail.stops != 0 {
		t.Fatalf("fail start=%d stop=%d", fail.starts, fail.stops)
	}
}
