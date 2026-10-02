package kernel_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel"
	zhttp "github.com/zatrano/framework/v3/core/kernel/http"
)

func assertHTTPUnavailable(t *testing.T, er zhttp.ExchangeResult) {
	t.Helper()
	if er.Status != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%q", er.Status, er.Body)
	}
	body := string(er.Body)
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

func TestHandleCreatedRouteDoesNotDispatch(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	hit := false
	app.Router().Get("/early", func(req *zhttp.Request) *zhttp.Response {
		hit = true
		return zhttp.Text("early")
	})
	er := serveHandle(t, app, http.MethodGet, "/early", nil, "")
	assertHTTPUnavailable(t, er)
	if hit {
		t.Fatal("Created dispatched the route handler")
	}
	if string(er.Body) == "early" {
		t.Fatal("Created served handler body")
	}
}

func TestHandleCreatedPublicFileIsNotRead(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "public", "app.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	er := serveHandle(t, app, http.MethodGet, "/app.css", nil, "")
	assertHTTPUnavailable(t, er)
	if string(er.Body) == "body{}" {
		t.Fatal("Created served a public file")
	}
}

func TestHandleBootFailedDoesNotDispatch(t *testing.T) {
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
	er := serveHandle(t, app, http.MethodGet, "/secret", nil, "")
	assertHTTPUnavailable(t, er)
	if hit {
		t.Fatal("BootFailed dispatched a pre-registered route")
	}
	if strings.Contains(string(er.Body), "secret") {
		t.Fatal("BootFailed served handler body")
	}
}

func TestHandleBootedWithoutStartAccepts(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	er := serveHandle(t, app, http.MethodGet, "/ok", nil, "")
	if er.Status != 200 || string(er.Body) != "ok" {
		t.Fatalf("Booted Handle: status=%d body=%q", er.Status, er.Body)
	}
	if er.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("Booted Handle skipped kernel middleware")
	}
}

func TestHandleProductionRejectionDoesNotLeakState(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	app := kernel.NewApplication(t.TempDir())
	er := serveHandle(t, app, http.MethodGet, "/ok", nil, "")
	assertHTTPUnavailable(t, er)
}
