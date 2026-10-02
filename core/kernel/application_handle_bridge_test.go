package kernel_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/contracts"
	"github.com/zatrano/framework/v3/core/kernel"
	zhttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
)

type taggedBridge struct {
	tag string
}

func (b taggedBridge) Middleware() []any {
	tag := b.tag
	return []any{
		func(next routing.HandlerFunc) routing.HandlerFunc {
			return func(req *zhttp.Request) *zhttp.Response {
				return next(req).Header("X-Bridge-MW", tag)
			}
		},
	}
}

func (b taggedBridge) Finalize(req any, resp any) any {
	if r, ok := resp.(*zhttp.Response); ok && r != nil {
		return r.Header("X-Bridge-Fin", b.tag)
	}
	return resp
}

func serveBridge(t *testing.T, app *kernel.Application) zhttp.ExchangeResult {
	t.Helper()
	return serveHandle(t, app, http.MethodGet, "/ok", nil, "")
}

func assertBridge(t *testing.T, er zhttp.ExchangeResult, tag string) {
	t.Helper()
	if er.Status != 200 || string(er.Body) != "ok" {
		t.Fatalf("status=%d body=%q", er.Status, er.Body)
	}
	if er.Header.Get("X-Bridge-MW") != tag {
		t.Fatalf("MW=%q want %s", er.Header.Get("X-Bridge-MW"), tag)
	}
	if er.Header.Get("X-Bridge-Fin") != tag {
		t.Fatalf("Finalize=%q want %s", er.Header.Get("X-Bridge-Fin"), tag)
	}
}

func TestHTTPBridgeCreatedSetCapturedTogether(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
	})
	app.SetHTTPBridge(taggedBridge{tag: "A"})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	assertBridge(t, serveBridge(t, app), "A")
}

type registerBridgeProvider struct{ tag string }

func (p registerBridgeProvider) Register(app contracts.App) error {
	app.SetHTTPBridge(taggedBridge{tag: p.tag})
	return nil
}
func (p registerBridgeProvider) Boot(app contracts.App) error { return nil }

func TestHTTPBridgeRegisterSetCapturedTogether(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
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
	app.Router().Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
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
	app.Router().Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
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
	app.Router().Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
	})
	app.RegisterProviders(bootSwapBridgeProvider{})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	assertBridge(t, serveBridge(t, app), "REG")
}

func bootBridgeApp(t *testing.T, bridge contracts.HTTPBridge) *kernel.Application {
	t.Helper()
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		req.Cookies().Queue("sid", "1", 10)
		return zhttp.Text("handler")
	})
	app.SetHTTPBridge(bridge)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	return app
}

type returnOnlyBridge struct{}

func (returnOnlyBridge) Middleware() []any { return nil }
func (returnOnlyBridge) Finalize(req any, resp any) any {
	if r, ok := resp.(*zhttp.Response); ok && r != nil {
		return r.Header("X-Fin-Resp", "1")
	}
	return resp
}

func TestFinalizeReturnOnlyKeepsBodyAndCookies(t *testing.T) {
	app := bootBridgeApp(t, returnOnlyBridge{})
	er := serveHandle(t, app, http.MethodGet, "/ok", nil, "")
	if er.Status != 200 || string(er.Body) != "handler" {
		t.Fatalf("return-only: status=%d body=%q", er.Status, er.Body)
	}
	if er.Header.Get("X-Fin-Resp") != "1" {
		t.Fatal("return-only dropped response header")
	}
	if er.Header.Get("Set-Cookie") == "" {
		t.Fatal("return-only dropped queued cookies")
	}
}

type nilReturnBridge struct{}

func (nilReturnBridge) Middleware() []any { return nil }
func (nilReturnBridge) Finalize(req any, resp any) any {
	return nil
}

func TestFinalizeNilReturnWithoutCommitUsesOriginal(t *testing.T) {
	app := bootBridgeApp(t, nilReturnBridge{})
	er := serveHandle(t, app, http.MethodGet, "/ok", nil, "")
	if er.Status != 200 || string(er.Body) != "handler" {
		t.Fatalf("nil return: status=%d body=%q", er.Status, er.Body)
	}
}
