package kernel_test

import (
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel"
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/routing"
)

type recoverFixture struct{}

func (p *recoverFixture) Register(app contracts.App) error {
	r := routing.From(app)
	r.Use(func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			switch req.Path() {
			case "/mw-panic":
				panic("middleware boom")
			case "/after-mw-panic":
				_ = next(req)
				panic("cookie-stage boom")
			}
			return next(req)
		}
	})
	r.Get("/handler-panic", func(req *http.Request) *http.Response {
		panic("handler boom")
	})
	r.Get("/mw-panic", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	r.Get("/after-mw-panic", func(req *http.Request) *http.Response {
		req.Cookies().Queue("sid", "1", 10)
		return http.Text("ok")
	})
	r.Get("/writeto-panic", func(req *http.Request) *http.Response {
		return http.Hijack(func(w stdhttp.ResponseWriter) error {
			panic("writeto boom")
		})
	})
	r.Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	return nil
}

func (p *recoverFixture) Boot(app contracts.App) error { return nil }

func bootRecoverApp(t *testing.T) *kernel.Application {
	t.Helper()
	return bootRecoverAppWithBridge(t, nil)
}

func bootRecoverAppWithBridge(t *testing.T, bridge contracts.HTTPBridge) *kernel.Application {
	t.Helper()
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.RegisterProviders(&recoverFixture{})
	if bridge != nil {
		app.SetHTTPBridge(bridge)
	}
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	return app
}

func serveRecover(t *testing.T, app *kernel.Application, req *stdhttp.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func assertRecovered500(t *testing.T, rec *httptest.ResponseRecorder, secret string) {
	t.Helper()
	if rec.Code != stdhttp.StatusInternalServerError {
		t.Fatalf("status=%d want 500", rec.Code)
	}
	body := rec.Body.String()
	if secret != "" && strings.Contains(body, secret) {
		t.Fatalf("panic leaked: %s", body)
	}
}

func TestTHR01HandlerPanic(t *testing.T) {
	app := bootRecoverApp(t)
	rec := serveRecover(t, app, httptest.NewRequest(stdhttp.MethodGet, "/handler-panic", nil))
	if rec.Code != stdhttp.StatusInternalServerError {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestTHR02MiddlewarePanic(t *testing.T) {
	app := bootRecoverApp(t)
	rec := serveRecover(t, app, httptest.NewRequest(stdhttp.MethodGet, "/mw-panic", nil))
	assertRecovered500(t, rec, "middleware boom")
}

func TestTHR03NewRequestPanic(t *testing.T) {
	app := bootRecoverApp(t)
	req := &stdhttp.Request{Method: stdhttp.MethodGet}
	rec := serveRecover(t, app, req)
	assertRecovered500(t, rec, "")
}

func TestTHR04MethodOverrideParseFormPanic(t *testing.T) {
	app := bootRecoverApp(t)
	req := httptest.NewRequest(stdhttp.MethodPost, "/ok", panicReader{})
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := serveRecover(t, app, req)
	assertRecovered500(t, rec, "parseform boom")
}

type panicReader struct{}

func (panicReader) Read([]byte) (int, error) { panic("parseform boom") }
func (panicReader) Close() error             { return nil }

type panicBridge struct{}

func (panicBridge) Middleware() []any { return nil }
func (panicBridge) Finalize(w stdhttp.ResponseWriter, req any, resp any) any {
	panic("finalize boom")
}

func TestTHR05FinalizePanic(t *testing.T) {
	app := bootRecoverAppWithBridge(t, panicBridge{})
	rec := serveRecover(t, app, httptest.NewRequest(stdhttp.MethodGet, "/ok", nil))
	assertRecovered500(t, rec, "finalize boom")
}

type commitThenPanicBridge struct{}

func (commitThenPanicBridge) Middleware() []any { return nil }
func (commitThenPanicBridge) Finalize(w stdhttp.ResponseWriter, req any, resp any) any {
	w.WriteHeader(stdhttp.StatusOK)
	_, _ = io.WriteString(w, "partial")
	panic("finalize after commit")
}

func TestTHR05FinalizePanicAfterCommitKeepsStatus(t *testing.T) {
	app := bootRecoverAppWithBridge(t, commitThenPanicBridge{})
	rec := serveRecover(t, app, httptest.NewRequest(stdhttp.MethodGet, "/ok", nil))
	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("committed status rewritten: %d", rec.Code)
	}
	if rec.Body.String() != "partial" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestTHR06CookieApplicationPanic(t *testing.T) {
	app := bootRecoverApp(t)
	rec := serveRecover(t, app, httptest.NewRequest(stdhttp.MethodGet, "/after-mw-panic", nil))
	assertRecovered500(t, rec, "cookie-stage boom")
}

func TestTHR07WriteToPanic(t *testing.T) {
	app := bootRecoverApp(t)
	rec := serveRecover(t, app, httptest.NewRequest(stdhttp.MethodGet, "/writeto-panic", nil))
	assertRecovered500(t, rec, "writeto boom")
}

type panicOnBodyWriter struct {
	stdhttp.ResponseWriter
}

func (w *panicOnBodyWriter) Write(p []byte) (int, error) {
	panic("static write boom")
}

func TestTHR08StaticWritePanicAfterCommit(t *testing.T) {
	app := bootPublicApp(t)
	rec := httptest.NewRecorder()
	wrapped := &panicOnBodyWriter{ResponseWriter: rec}
	app.ServeHTTP(wrapped, httptest.NewRequest(stdhttp.MethodGet, "/app.css", nil))
	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("committed static status rewritten: %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "static write boom") {
		t.Fatalf("panic leaked: %s", rec.Body.String())
	}
}

type panicOnWriteHeader struct {
	header stdhttp.Header
}

func (w *panicOnWriteHeader) Header() stdhttp.Header {
	if w.header == nil {
		w.header = make(stdhttp.Header)
	}
	return w.header
}
func (w *panicOnWriteHeader) Write([]byte) (int, error) { panic("recovery write") }
func (w *panicOnWriteHeader) WriteHeader(int)           { panic("recovery writeheader") }

func TestRecoveredWritePanicDoesNotEscape(t *testing.T) {
	app := bootRecoverApp(t)
	req := &stdhttp.Request{Method: stdhttp.MethodGet}
	app.ServeHTTP(&panicOnWriteHeader{}, req)
}
