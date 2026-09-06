package kernel_test

import (
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

func snapshotHasPath(app *kernel.Application, path string) bool {
	for _, r := range app.Router().Snapshot() {
		if r.Path == path {
			return true
		}
	}
	return false
}

func TestGetTypedHandlerFuncRegisters(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	var h routing.HandlerFunc = func(req *http.Request) *http.Response {
		return http.Text("ok")
	}
	app.Router().Get("/ok", h)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/ok", nil))
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestGetFuncHandlerRegisters(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/ok", nil))
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestGetUnsupportedHandlerPanicsAndDoesNotRegister(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("unsupported handler was registered")
		}
		if !strings.Contains(fmt.Sprint(rec), "unsupported handler") {
			t.Fatalf("panic=%v", rec)
		}
		if snapshotHasPath(app, "/bad") {
			t.Fatal("unsupported handler entered the route table")
		}
	}()
	app.Router().Get("/bad", "not-a-handler")
}

func TestGetUnsupportedIntPanics(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	defer func() {
		if recover() == nil {
			t.Fatal("int handler was registered")
		}
	}()
	app.Router().Get("/bad", 42)
}

type namedBadHandler struct{}

func TestGetNamedUnsupportedTypePanics(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("named type handler was registered")
		}
		if !strings.Contains(fmt.Sprint(rec), "unsupported handler") {
			t.Fatalf("panic=%v", rec)
		}
	}()
	app.Router().Get("/bad", namedBadHandler{})
}

func TestGetNilHandlerPanicsAndDoesNotRegister(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	defer func() {
		if recover() == nil {
			t.Fatal("nil handler was registered")
		}
		if snapshotHasPath(app, "/nil") {
			t.Fatal("nil handler entered the route table")
		}
	}()
	app.Router().Get("/nil", nil)
}

func TestGetTypedNilHandlerFuncPanics(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	defer func() {
		if recover() == nil {
			t.Fatal("typed nil HandlerFunc was registered")
		}
		if snapshotHasPath(app, "/nilfn") {
			t.Fatal("typed nil HandlerFunc entered the route table")
		}
	}()
	var h routing.HandlerFunc
	app.Router().Get("/nilfn", h)
}

func TestPostUnsupportedHandlerPanicsAndDoesNotRegister(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	defer func() {
		if recover() == nil {
			t.Fatal("unsupported POST handler was registered")
		}
		if snapshotHasPath(app, "/bad") {
			t.Fatal("unsupported POST handler entered the route table")
		}
	}()
	app.Router().Post("/bad", "not-a-handler")
}

func TestUnsupportedHandlerDoesNotServe500TypeLeak(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	func() {
		defer func() { _ = recover() }()
		app.Router().Get("/leak", "not-a-handler")
	}()
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if snapshotHasPath(app, "/leak") {
		t.Fatal("unsupported handler entered the route table")
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/leak", nil))
	if rec.Code == 500 {
		t.Fatalf("request-time 500: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "unsupported handler") {
		t.Fatalf("type leaked: %s", rec.Body.String())
	}
}

func TestTypedRouterFacadeUnchanged(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	r := routing.From(app)
	r.Get("/typed", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/typed", nil))
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("typed facade broken: status=%d body=%q", rec.Code, rec.Body.String())
	}
}
