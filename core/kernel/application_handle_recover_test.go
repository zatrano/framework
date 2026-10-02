package kernel_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/contracts"
	"github.com/zatrano/framework/v3/core/kernel"
	zhttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
)

type recoverFixture struct{}

func (p *recoverFixture) Register(app contracts.App) error {
	r := routing.From(app)
	r.Use(func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *zhttp.Request) *zhttp.Response {
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
	r.Get("/handler-panic", func(req *zhttp.Request) *zhttp.Response {
		panic("handler boom")
	})
	r.Get("/mw-panic", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
	})
	r.Get("/after-mw-panic", func(req *zhttp.Request) *zhttp.Response {
		req.Cookies().Queue("sid", "1", 10)
		return zhttp.Text("ok")
	})
	r.Get("/ok", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.Text("ok")
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

func assertRecovered500(t *testing.T, er zhttp.ExchangeResult, secret string) {
	t.Helper()
	if er.Status != http.StatusInternalServerError {
		t.Fatalf("status=%d want 500 body=%q", er.Status, er.Body)
	}
	body := string(er.Body)
	if secret != "" && strings.Contains(body, secret) {
		t.Fatalf("panic leaked: %s", body)
	}
}

func TestTHR01HandlerPanic(t *testing.T) {
	app := bootRecoverApp(t)
	er := serveHandle(t, app, http.MethodGet, "/handler-panic", nil, "")
	assertRecovered500(t, er, "handler boom")
}

func TestTHR02MiddlewarePanic(t *testing.T) {
	app := bootRecoverApp(t)
	er := serveHandle(t, app, http.MethodGet, "/mw-panic", nil, "")
	assertRecovered500(t, er, "middleware boom")
}

type panicBridge struct{}

func (panicBridge) Middleware() []any { return nil }
func (panicBridge) Finalize(req any, resp any) any {
	panic("finalize boom")
}

func TestTHR05FinalizePanic(t *testing.T) {
	app := bootRecoverAppWithBridge(t, panicBridge{})
	er := serveHandle(t, app, http.MethodGet, "/ok", nil, "")
	assertRecovered500(t, er, "finalize boom")
}

func TestTHR06CookieApplicationPanic(t *testing.T) {
	app := bootRecoverApp(t)
	er := serveHandle(t, app, http.MethodGet, "/after-mw-panic", nil, "")
	assertRecovered500(t, er, "cookie-stage boom")
}
