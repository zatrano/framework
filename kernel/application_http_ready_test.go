package kernel_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/kernel"
	zhttp "github.com/zatrano/framework/kernel/http"
)

func assertHTTPUnavailable(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, leak := range []string{
		"Created", "Bootstrapping", "BootFailed", "lifeCreated", "lifeBoot",
		"application:", "refusing to boot", "APP_KEY", "unsupported handler",
		"%T", "provider",
	} {
		if strings.Contains(body, leak) {
			t.Fatalf("503 leaked %q: %s", leak, body)
		}
	}
}

func TestServeHTTPCreatedRouteDoesNotDispatch(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	hit := false
	app.Router().Get("/early", func(req *zhttp.Request) *zhttp.Response {
		hit = true
		return zhttp.Text("early")
	})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/early", nil))
	assertHTTPUnavailable(t, rec)
	if hit {
		t.Fatal("Created dispatched the route handler")
	}
	if rec.Body.String() == "early" {
		t.Fatal("Created served handler body")
	}
}

func TestServeHTTPCreatedPublicFileIsNotRead(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "public", "app.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.css", nil))
	assertHTTPUnavailable(t, rec)
	if rec.Body.String() == "body{}" {
		t.Fatal("Created served a public file")
	}
}

func TestServeHTTPBootstrappingDoesNotDispatch(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	entered := make(chan struct{})
	release := make(chan struct{})
	hit := false
	app.RegisterProviders(&blockRegisterProvider{
		entered: entered,
		release: release,
		hit:     &hit,
	})
	errCh := make(chan error, 1)
	go func() { errCh <- app.Bootstrap() }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("bootstrap did not enter Register")
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/during-boot", nil))
	close(release)
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	assertHTTPUnavailable(t, rec)
	if hit {
		t.Fatal("Bootstrapping dispatched a handler")
	}
}

func TestServeHTTPBootFailedDoesNotDispatch(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", "")
	app := kernel.NewApplication(t.TempDir())
	hit := false
	app.Router().Get("/secret", func(req *zhttp.Request) *zhttp.Response {
		hit = true
		return zhttp.Text("secret")
	})
	if err := app.Bootstrap(); err == nil {
		t.Fatal("expected bootstrap failure")
	}
	if !app.BootstrapFailed() {
		t.Fatal("expected BootstrapFailed")
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/secret", nil))
	assertHTTPUnavailable(t, rec)
	if hit {
		t.Fatal("BootFailed dispatched a pre-registered route")
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatal("BootFailed served handler body")
	}
}

func TestServeHTTPBootedWithoutStartAccepts(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("Booted ServeHTTP: status=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("Booted ServeHTTP skipped kernel middleware")
	}
}

func TestServeHTTPStartingAccepts(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	entered := make(chan struct{})
	release := make(chan struct{})
	app.RegisterProviders(&blockStartHTTPProvider{entered: entered, release: release})
	errCh := make(chan error, 1)
	go func() { errCh <- app.Start() }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not enter provider")
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ok", nil))
	close(release)
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("Starting ServeHTTP: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestServeHTTPRunningAccepts(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
	})
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Stop(context.Background()) })
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("Running ServeHTTP: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestServeHTTPStoppingAccepts(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	entered := make(chan struct{})
	release := make(chan struct{})
	app.RegisterProviders(&blockStopHTTPProvider{entered: entered, release: release})
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() { errCh <- app.Stop(context.Background()) }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not enter provider")
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ok", nil))
	close(release)
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("Stopping ServeHTTP: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestServeHTTPStoppedAccepts(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
	})
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("Stopped ServeHTTP: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestServeHTTPProductionRejectionDoesNotLeakState(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	app := kernel.NewApplication(t.TempDir())
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ok", nil))
	assertHTTPUnavailable(t, rec)
}

type blockRegisterProvider struct {
	entered chan struct{}
	release chan struct{}
	hit     *bool
}

func (p *blockRegisterProvider) Register(app contracts.App) error {
	app.Router().Get("/during-boot", func(req *zhttp.Request) *zhttp.Response {
		*p.hit = true
		return zhttp.Text("during-boot")
	})
	close(p.entered)
	<-p.release
	return nil
}

func (p *blockRegisterProvider) Boot(app contracts.App) error { return nil }

type blockStartHTTPProvider struct {
	entered chan struct{}
	release chan struct{}
}

func (p *blockStartHTTPProvider) Register(app contracts.App) error {
	app.Router().Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
	})
	return nil
}

func (p *blockStartHTTPProvider) Boot(app contracts.App) error { return nil }

func (p *blockStartHTTPProvider) Start(app contracts.App) error {
	close(p.entered)
	<-p.release
	return nil
}

func (p *blockStartHTTPProvider) Stop(ctx context.Context) error { return nil }

type blockStopHTTPProvider struct {
	entered chan struct{}
	release chan struct{}
}

func (p *blockStopHTTPProvider) Register(app contracts.App) error {
	app.Router().Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
	})
	return nil
}

func (p *blockStopHTTPProvider) Boot(app contracts.App) error { return nil }

func (p *blockStopHTTPProvider) Start(app contracts.App) error { return nil }

func (p *blockStopHTTPProvider) Stop(ctx context.Context) error {
	close(p.entered)
	<-p.release
	return nil
}
