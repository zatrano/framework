package reference

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestHTTPNotReadyBeforeBootstrap(t *testing.T) {
	app := Assemble(t.TempDir())
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/up", nil))
	if rec.Code != stdhttp.StatusServiceUnavailable {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHTTPUpAfterBootstrapWithoutStart(t *testing.T) {
	app := bootOnly(t)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/up", nil))
	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
	status := httptest.NewRecorder()
	app.ServeHTTP(status, httptest.NewRequest(stdhttp.MethodGet, "/api/v1/status", nil))
	if status.Code != stdhttp.StatusOK {
		t.Fatalf("status endpoint=%d", status.Code)
	}
	if strings.Contains(status.Body.String(), `"worker":true`) {
		t.Fatal("/api/v1/status must not report worker running before Start")
	}
}

func TestHTTPStatusAndItemAfterStart(t *testing.T) {
	secret := "super-secret-value-xyz"
	t.Setenv(envAPIToken, secret)
	t.Setenv(envName, "catalog")
	app := bootAndStart(t)

	up := httptest.NewRecorder()
	app.ServeHTTP(up, httptest.NewRequest(stdhttp.MethodGet, "/up", nil))
	if up.Code != stdhttp.StatusOK {
		t.Fatalf("/up=%d", up.Code)
	}

	status := httptest.NewRecorder()
	app.ServeHTTP(status, httptest.NewRequest(stdhttp.MethodGet, "/api/v1/status", nil))
	body := status.Body.String()
	if status.Code != stdhttp.StatusOK {
		t.Fatalf("/status=%d %s", status.Code, body)
	}
	if !strings.Contains(body, `"name":"catalog"`) {
		t.Fatalf("status body=%s", body)
	}
	if !strings.Contains(body, `"worker":true`) {
		t.Fatalf("worker should run after Start: %s", body)
	}
	if strings.Contains(body, secret) {
		t.Fatalf("token leaked in status: %s", body)
	}

	item := httptest.NewRecorder()
	app.ServeHTTP(item, httptest.NewRequest(stdhttp.MethodGet, "http://example/api/v1/items/1", nil))
	if item.Code != stdhttp.StatusOK || !strings.Contains(item.Body.String(), "alpha") {
		t.Fatalf("item=%d %s", item.Code, item.Body.String())
	}

	missing := httptest.NewRecorder()
	app.ServeHTTP(missing, httptest.NewRequest(stdhttp.MethodGet, "http://example/api/v1/items/nope", nil))
	if missing.Code != stdhttp.StatusNotFound {
		t.Fatalf("missing=%d %s", missing.Code, missing.Body.String())
	}
	if strings.Contains(missing.Body.String(), secret) {
		t.Fatalf("token leaked in 404: %s", missing.Body.String())
	}
}

func bootOnly(t *testing.T, extra ...kernel.Provider) *kernel.Application {
	t.Helper()
	app := Assemble(t.TempDir(), extra...)
	t.Cleanup(func() { closeLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	return app
}

func bootAndStart(t *testing.T, extra ...kernel.Provider) *kernel.Application {
	t.Helper()
	app := bootOnly(t, extra...)
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Stop(context.Background()) })
	return app
}

func closeLog(t *testing.T, app *kernel.Application) {
	t.Helper()
	if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
		_ = c.Close()
	}
}
