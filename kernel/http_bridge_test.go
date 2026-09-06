package kernel_test

import (
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

type taggedBridge struct {
	tag string
}

func (b taggedBridge) Middleware() []any {
	tag := b.tag
	return []any{
		func(next routing.HandlerFunc) routing.HandlerFunc {
			return func(req *http.Request) *http.Response {
				return next(req).Header("X-Bridge-MW", tag)
			}
		},
	}
}

func (b taggedBridge) Finalize(w stdhttp.ResponseWriter, req any, resp any) any {
	if r, ok := resp.(*http.Response); ok && r != nil {
		return r.Header("X-Bridge-Fin", b.tag)
	}
	return resp
}

func serveBridge(t *testing.T, app *kernel.Application) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/ok", nil))
	return rec
}

func assertBridge(t *testing.T, rec *httptest.ResponseRecorder, tag string) {
	t.Helper()
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Bridge-MW") != tag {
		t.Fatalf("MW=%q want %s", rec.Header().Get("X-Bridge-MW"), tag)
	}
	if rec.Header().Get("X-Bridge-Fin") != tag {
		t.Fatalf("Finalize=%q want %s", rec.Header().Get("X-Bridge-Fin"), tag)
	}
}

func TestHTTPBridgeCreatedSetCapturedTogether(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	app.SetHTTPBridge(taggedBridge{tag: "A"})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	assertBridge(t, serveBridge(t, app), "A")
}

type registerBridgeProvider struct{ tag string }

func (p registerBridgeProvider) Register(app contracts.App) error {
	app.SetHTTPBridge(taggedBridge(p))
	return nil
}
func (p registerBridgeProvider) Boot(app contracts.App) error { return nil }

func TestHTTPBridgeRegisterSetCapturedTogether(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	app.RegisterProviders(registerBridgeProvider{tag: "R"})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	assertBridge(t, serveBridge(t, app), "R")
}

func TestHTTPBridgeSetAfterBootstrapPanicsAndKeepsCapture(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	app.SetHTTPBridge(taggedBridge{tag: "A"})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			rec := recover()
			if rec == nil {
				t.Fatal("SetHTTPBridge after bootstrap did not panic")
			}
			if !strings.Contains(fmt.Sprint(rec), "HTTP bridge") {
				t.Fatalf("panic=%v", rec)
			}
		}()
		app.SetHTTPBridge(taggedBridge{tag: "B"})
	}()
	assertBridge(t, serveBridge(t, app), "A")
}

func TestHTTPBridgeNilAfterBootstrapPanicsAndKeepsFinalize(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	app.SetHTTPBridge(taggedBridge{tag: "A"})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("SetHTTPBridge(nil) after bootstrap did not panic")
			}
		}()
		app.SetHTTPBridge(nil)
	}()
	assertBridge(t, serveBridge(t, app), "A")
}

type bootSwapBridgeProvider struct{}

func (p bootSwapBridgeProvider) Register(app contracts.App) error {
	app.SetHTTPBridge(taggedBridge{tag: "REG"})
	return nil
}

func (p bootSwapBridgeProvider) Boot(app contracts.App) error {
	defer func() {
		if recover() == nil {
			panic("SetHTTPBridge in Boot did not panic")
		}
	}()
	app.SetHTTPBridge(taggedBridge{tag: "BOOT"})
	return nil
}

func TestHTTPBridgeSetDuringBootPanicsAndKeepsRegisterCapture(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	app.RegisterProviders(bootSwapBridgeProvider{})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	assertBridge(t, serveBridge(t, app), "REG")
}
