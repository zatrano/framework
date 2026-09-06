package kernel_test

import (
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/contracts"
	"github.com/zatrano/framework/v2/kernel"
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/routing"
)

func TestUseUnsupportedMiddlewarePanics(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("unsupported middleware was silently ignored")
		}
		msg := fmt.Sprint(rec)
		if !strings.Contains(msg, "unsupported middleware") {
			t.Fatalf("panic=%v", rec)
		}
	}()
	app.Router().Use("not-a-middleware")
}

func TestUseNilMiddlewarePanics(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	defer func() {
		if recover() == nil {
			t.Fatal("nil middleware was silently ignored")
		}
	}()
	app.Router().Use(nil)
}

func TestGroupUnsupportedMiddlewarePanics(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	defer func() {
		if recover() == nil {
			t.Fatal("group middleware was silently ignored")
		}
	}()
	app.Router().Group("/api", func(contracts.Router) {}, 42)
}

type badMWBridge struct{}

func (badMWBridge) Middleware() []any { return []any{"not-mw"} }
func (badMWBridge) Finalize(w stdhttp.ResponseWriter, req any, resp any) any {
	return resp
}

func TestHTTPBridgeUnsupportedMiddlewareFailsBootstrap(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.SetHTTPBridge(badMWBridge{})
	if err := app.Bootstrap(); err == nil {
		t.Fatal("bootstrap accepted unsupported HTTPBridge middleware")
	}
}

func TestUseTypedMiddlewareStillRegisters(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	saw := false
	app.Router().Use(func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			saw = true
			return next(req)
		}
	})
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/ok", nil))
	if rec.Code != 200 || rec.Body.String() != "ok" || !saw {
		t.Fatalf("typed middleware dropped: status=%d body=%q saw=%v", rec.Code, rec.Body.String(), saw)
	}
}
