package kernel_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel"
	"github.com/zatrano/framework/v2/kernel/config"
	"github.com/zatrano/framework/v2/kernel/container"
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/routing"
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
	starts   int
	stops    int
	order    *[]string
	name     string
	failOnce error
	fail     error
}

func (p *lifecycleProbe) Register(app contracts.App) error { return nil }
func (p *lifecycleProbe) Boot(app contracts.App) error     { return nil }
func (p *lifecycleProbe) Start(app contracts.App) error {
	if p.order != nil {
		*p.order = append(*p.order, "start:"+p.name)
	}
	p.starts++
	if p.failOnce != nil && p.starts == 1 {
		return p.failOnce
	}
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

func TestLifecycleConcurrentStartOnce(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(p)
	var wg sync.WaitGroup
	errCh := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- app.Start()
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	if p.starts != 1 {
		t.Fatalf("concurrent Start launched worker %d times", p.starts)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p.stops != 1 {
		t.Fatalf("stops=%d", p.stops)
	}
}

func TestLifecycleConcurrentStopOnce(t *testing.T) {
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

func TestLifecycleConcurrentStartAndStop(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(p)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = app.Start()
	}()
	go func() {
		defer wg.Done()
		_ = app.Stop(context.Background())
	}()
	wg.Wait()
	if p.starts > 1 {
		t.Fatalf("Start ran %d times", p.starts)
	}
	if p.stops > p.starts {
		t.Fatalf("stops=%d starts=%d", p.stops, p.starts)
	}
}

func TestLifecycleFailedStartThenRetry(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	ok := &lifecycleProbe{name: "ok"}
	flaky := &lifecycleProbe{name: "flaky", failOnce: errors.New("first")}
	app.RegisterProviders(ok, flaky)
	if err := app.Start(); err == nil {
		t.Fatal("expected first start to fail")
	}
	if ok.starts != 1 || ok.stops != 1 {
		t.Fatalf("ok start=%d stop=%d", ok.starts, ok.stops)
	}
	if flaky.starts != 1 || flaky.stops != 0 {
		t.Fatalf("flaky start=%d stop=%d", flaky.starts, flaky.stops)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ok.stops != 1 {
		t.Fatalf("Stop while Booted must not double-stop, stops=%d", ok.stops)
	}
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if ok.starts != 2 || flaky.starts != 2 {
		t.Fatalf("retry starts ok=%d flaky=%d", ok.starts, flaky.starts)
	}
	if flaky.stops != 0 {
		t.Fatalf("failed first start must not have been stopped, stops=%d", flaky.stops)
	}
}

func TestLifecycleStartFailureConcurrentStop(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	a := &lifecycleProbe{name: "a"}
	b := &lifecycleProbe{name: "b"}
	c := &lifecycleProbe{name: "c", fail: errors.New("nope")}
	app.RegisterProviders(a, b, c)
	var wg sync.WaitGroup
	errCh := make(chan error, 1)
	wg.Add(2)
	go func() {
		defer wg.Done()
		errCh <- app.Start()
	}()
	go func() {
		defer wg.Done()
		_ = app.Stop(context.Background())
	}()
	wg.Wait()
	startErr := <-errCh
	if startErr == nil {
		t.Fatal("expected start failure")
	}
	if a.starts != 1 || b.starts != 1 || c.starts != 1 {
		t.Fatalf("starts a=%d b=%d c=%d", a.starts, b.starts, c.starts)
	}
	if a.stops != 1 || b.stops != 1 || c.stops != 0 {
		t.Fatalf("stops a=%d b=%d c=%d", a.stops, b.stops, c.stops)
	}
}

// gateProvider blocks in Register so tests can overlap Bootstrap and Start.
type gateProvider struct {
	entered   chan struct{}
	release   chan struct{}
	enterOnce sync.Once
	registers int
	boots     int
}

func (p *gateProvider) Register(app contracts.App) error {
	p.registers++
	p.enterOnce.Do(func() {
		if p.entered != nil {
			close(p.entered)
		}
	})
	if p.release != nil {
		<-p.release
	}
	return nil
}

func (p *gateProvider) Boot(app contracts.App) error {
	p.boots++
	return nil
}

func TestLifecycleConcurrentBootstrapAndStart(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	entered := make(chan struct{})
	release := make(chan struct{})
	gate := &gateProvider{entered: entered, release: release}
	worker := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(gate, worker)

	bootErr := make(chan error, 1)
	go func() { bootErr <- app.Bootstrap() }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("Bootstrap never entered Register")
	}

	startErr := make(chan error, 1)
	go func() { startErr <- app.Start() }()

	select {
	case err := <-startErr:
		t.Fatalf("Start must wait for in-progress Bootstrap, got %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(release)

	select {
	case err := <-bootErr:
		if err != nil {
			t.Fatalf("Bootstrap: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Bootstrap hung")
	}
	select {
	case err := <-startErr:
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start hung")
	}

	if gate.registers != 1 || gate.boots != 1 {
		t.Fatalf("register=%d boot=%d", gate.registers, gate.boots)
	}
	if worker.starts != 1 {
		t.Fatalf("starts=%d", worker.starts)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleConcurrentStartThenBootstrap(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	entered := make(chan struct{})
	release := make(chan struct{})
	gate := &gateProvider{entered: entered, release: release}
	app.RegisterProviders(gate)

	startErr := make(chan error, 1)
	go func() { startErr <- app.Start() }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("Start never entered Register")
	}

	bootErr := make(chan error, 1)
	go func() { bootErr <- app.Bootstrap() }()

	select {
	case err := <-bootErr:
		t.Fatalf("Bootstrap must wait for in-progress Start, got %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(release)

	select {
	case err := <-startErr:
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start hung")
	}
	select {
	case err := <-bootErr:
		if err != nil {
			t.Fatalf("Bootstrap: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Bootstrap hung")
	}
	if gate.registers != 1 || gate.boots != 1 {
		t.Fatalf("register=%d boot=%d", gate.registers, gate.boots)
	}
}

func TestLifecycleConcurrentBootstrapOnce(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	entered := make(chan struct{})
	release := make(chan struct{})
	gate := &gateProvider{entered: entered, release: release}
	app.RegisterProviders(gate)

	first := make(chan error, 1)
	go func() { first <- app.Bootstrap() }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first Bootstrap never entered Register")
	}

	second := make(chan error, 1)
	go func() { second <- app.Bootstrap() }()

	select {
	case err := <-second:
		t.Fatalf("second Bootstrap must wait, got %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(release)

	for _, ch := range []chan error{first, second} {
		select {
		case err := <-ch:
			if err != nil {
				t.Fatalf("Bootstrap: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Bootstrap hung")
		}
	}
	if gate.registers != 1 || gate.boots != 1 {
		t.Fatalf("register=%d boot=%d", gate.registers, gate.boots)
	}
}
