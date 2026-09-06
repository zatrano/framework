package tests

import (
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/bootstrap"
	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/routing"
)

const exceptionPathSecret = "hunter2-exception-path-secret"

type exceptionPathFixture struct{}

func (p *exceptionPathFixture) Register(app contracts.App) error {
	r := routing.From(app)
	r.Get("/handler-panic", func(req *http.Request) *http.Response {
		panic(exceptionPathSecret)
	})
	r.Get("/mw-panic", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	return nil
}

func (p *exceptionPathFixture) Boot(app contracts.App) error {
	// Kernel installs exceptions.Handler after Register(). Middleware added
	// here sits inside that chain; Register()-time Use sits outside it.
	r := routing.From(app)
	r.Use(func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if req.Path() == "/mw-panic" {
				panic(exceptionPathSecret)
			}
			return next(req)
		}
	})
	return nil
}

func bootExceptionPathApp(t *testing.T) contracts.App {
	t.Helper()
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	app := bootstrap.App(bootstrap.WithProviders(&exceptionPathFixture{}))
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.Exceptions() == nil {
		t.Fatal("expected exceptions.Handler from KernelServiceProvider, not middleware.Recover fallback")
	}
	if app.IsDebug() {
		t.Fatal("APP_DEBUG=false must load into boot config; IsDebug() true means this is not the production renderer")
	}
	if app.Config().GetBool("app.debug", true) {
		t.Fatal("app.debug must come from boot configuration as false")
	}
	if app.Reports() == nil {
		t.Fatal("expected Reports from KernelServiceProvider")
	}
	t.Cleanup(func() {
		if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
			_ = c.Close()
		}
	})
	return app
}

func serveExceptionPath(t *testing.T, app contracts.App, path, accept string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(stdhttp.MethodGet, path, nil)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func assertProductionJSON(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != stdhttp.StatusInternalServerError {
		t.Fatalf("status=%d want 500", rec.Code)
	}
	body := rec.Body.String()
	assertNoExceptionLeak(t, body)
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("JSON: %v body=%s", err, body)
	}
	if payload["message"] != "Server Error" {
		t.Fatalf("message=%v", payload["message"])
	}
	if _, ok := payload["exception"]; ok {
		t.Fatalf("exception field present: %#v", payload)
	}
	if len(payload) != 1 {
		t.Fatalf("unexpected JSON keys: %#v", payload)
	}
}

func assertProductionHTML(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != stdhttp.StatusInternalServerError {
		t.Fatalf("status=%d want 500", rec.Code)
	}
	body := rec.Body.String()
	assertNoExceptionLeak(t, body)
	if !strings.Contains(body, "Server Error") {
		t.Fatalf("HTML missing Server Error: %s", body)
	}
}

func assertNoExceptionLeak(t *testing.T, body string) {
	t.Helper()
	for _, leak := range []string{exceptionPathSecret, "goroutine ", ".go:", "runtime/debug"} {
		if strings.Contains(body, leak) {
			t.Fatalf("leaked %q: %s", leak, body)
		}
	}
}

func assertReportedPanic(t *testing.T, app contracts.App, path string) {
	t.Helper()
	reports := app.Reports()
	if reports == nil {
		t.Fatal("Reports() is nil")
	}
	recent := reports.Recent(1)
	if len(recent) == 0 {
		t.Fatal("Reports().Recent() empty after Handler panic")
	}
	ev := recent[0]
	if !strings.Contains(ev.Message, exceptionPathSecret) {
		t.Fatalf("report message=%q want %q", ev.Message, exceptionPathSecret)
	}
	if ev.Method != stdhttp.MethodGet {
		t.Fatalf("report method=%q", ev.Method)
	}
	if ev.Path != path {
		t.Fatalf("report path=%q want %q", ev.Path, path)
	}
}

func TestProductionHandlerPanicJSON(t *testing.T) {
	app := bootExceptionPathApp(t)
	rec := serveExceptionPath(t, app, "/handler-panic", "application/json")
	assertProductionJSON(t, rec)
	assertReportedPanic(t, app, "/handler-panic")
}

func TestProductionHandlerPanicHTML(t *testing.T) {
	app := bootExceptionPathApp(t)
	rec := serveExceptionPath(t, app, "/handler-panic", "text/html")
	assertProductionHTML(t, rec)
	assertReportedPanic(t, app, "/handler-panic")
}

func TestProductionMiddlewarePanicJSON(t *testing.T) {
	app := bootExceptionPathApp(t)
	rec := serveExceptionPath(t, app, "/mw-panic", "application/json")
	assertProductionJSON(t, rec)
	assertReportedPanic(t, app, "/mw-panic")
}

func TestProductionMiddlewarePanicHTML(t *testing.T) {
	app := bootExceptionPathApp(t)
	rec := serveExceptionPath(t, app, "/mw-panic", "text/html")
	assertProductionHTML(t, rec)
	assertReportedPanic(t, app, "/mw-panic")
}
