package kernel_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/zatrano/framework/v2/kernel"
	zhttp "github.com/zatrano/framework/v2/kernel/http"
)

// Phase 11 Contract B locks the existing state machine through public APIs.
// There is no exported lifecycle type; Bootstrapped / BootstrapFailed / HTTP
// readiness / Start+Stop behavior are the contract.

func TestLifecycleContractCreatedIsNotHTTPReady(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	if app.Bootstrapped() || app.BootstrapFailed() {
		t.Fatal("Created must not be Bootstrapped or BootstrapFailed")
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("Created HTTP status=%d", rec.Code)
	}
}

func TestLifecycleContractBootstrappingIsNotHTTPReady(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	entered := make(chan struct{})
	release := make(chan struct{})
	app.RegisterProviders(&gateProvider{entered: entered, release: release})
	errCh := make(chan error, 1)
	go func() { errCh <- app.Bootstrap() }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("never entered Register")
	}
	if app.Bootstrapped() || app.BootstrapFailed() {
		t.Fatal("Bootstrapping must not report Bootstrapped or BootstrapFailed")
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	close(release)
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("Bootstrapping HTTP status=%d", rec.Code)
	}
}

func TestLifecycleContractBootFailedIsTerminalAndNotHTTPReady(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.RegisterProviders(&failRegisterProvider{})
	if err := app.Bootstrap(); err == nil {
		t.Fatal("expected Bootstrap error")
	}
	if !app.BootstrapFailed() || app.Bootstrapped() {
		t.Fatal("expected BootFailed")
	}
	if err := app.Bootstrap(); err == nil {
		t.Fatal("Bootstrap failure is terminal")
	}
	if err := app.Start(); err == nil {
		t.Fatal("Start after BootFailed must fail")
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("BootFailed HTTP status=%d", rec.Code)
	}
}

func TestLifecycleContractHTTPReadyBeginsAtBooted(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	worker := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(worker)
	app.Router().Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if !app.Bootstrapped() || app.BootstrapFailed() {
		t.Fatal("expected Booted")
	}
	if worker.starts != 0 {
		t.Fatalf("process workers must start only while Running, starts=%d", worker.starts)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("Booted HTTP status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestLifecycleContractStartFailureReturnsToBootedAndRetry(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	ok := &lifecycleProbe{name: "ok"}
	flaky := &lifecycleProbe{name: "flaky", failOnce: errors.New("first")}
	app.RegisterProviders(ok, flaky)
	if err := app.Start(); err == nil {
		t.Fatal("expected Start failure")
	}
	if !app.Bootstrapped() || app.BootstrapFailed() {
		t.Fatal("Start failure must leave the application Booted")
	}
	if ok.starts != 1 || ok.stops != 1 || flaky.stops != 0 {
		t.Fatalf("cleanup started-only: ok start=%d stop=%d flaky stop=%d", ok.starts, ok.stops, flaky.stops)
	}
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if ok.starts != 2 || flaky.starts != 2 {
		t.Fatalf("retry starts ok=%d flaky=%d", ok.starts, flaky.starts)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleContractStoppedCannotStart(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(p)
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if p.starts != 1 {
		t.Fatalf("workers must run only after Start, starts=%d", p.starts)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := app.Start(); err == nil {
		t.Fatal("Stopped cannot Start again")
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code == http.StatusServiceUnavailable {
		t.Fatal("Stopped remains HTTP-ready")
	}
}

func TestLifecycleContractWorkersOnlyAfterStart(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(p)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if p.starts != 0 {
		t.Fatal("Booted must not Start process workers")
	}
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if p.starts != 1 {
		t.Fatalf("Running starts=%d", p.starts)
	}
	_ = app.Stop(context.Background())
}

func TestLifecycleContractStartingAndStoppingStayHTTPReady(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	entered := make(chan struct{})
	release := make(chan struct{})
	app.RegisterProviders(&blockStartHTTPProvider{entered: entered, release: release})
	errCh := make(chan error, 1)
	go func() { errCh <- app.Start() }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("Start never entered")
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ok", nil))
	close(release)
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 {
		t.Fatalf("Starting HTTP status=%d", rec.Code)
	}

	stopEntered := make(chan struct{})
	stopRelease := make(chan struct{})
	app2 := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app2) })
	app2.RegisterProviders(&blockStopHTTPProvider{entered: stopEntered, release: stopRelease})
	if err := app2.Start(); err != nil {
		t.Fatal(err)
	}
	stopErr := make(chan error, 1)
	go func() { stopErr <- app2.Stop(context.Background()) }()
	select {
	case <-stopEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop never entered")
	}
	rec2 := httptest.NewRecorder()
	app2.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/ok", nil))
	close(stopRelease)
	if err := <-stopErr; err != nil {
		t.Fatal(err)
	}
	if rec2.Code != 200 {
		t.Fatalf("Stopping HTTP status=%d", rec2.Code)
	}
}

func TestLifecycleContractConcurrentStartOnce(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	p := &lifecycleProbe{name: "worker"}
	app.RegisterProviders(p)
	var wg sync.WaitGroup
	errCh := make(chan error, 4)
	for i := 0; i < 4; i++ {
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
		t.Fatalf("starts=%d", p.starts)
	}
	_ = app.Stop(context.Background())
}
