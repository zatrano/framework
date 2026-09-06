package kernel_test

import (
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/kernel/http"
)

func bootBridgeApp(t *testing.T, bridge contracts.HTTPBridge) *kernel.Application {
	t.Helper()
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		req.Cookies().Queue("sid", "1", 10)
		return http.Text("handler")
	})
	app.SetHTTPBridge(bridge)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	return app
}

func serveOK(t *testing.T, app *kernel.Application) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/ok", nil))
	return rec
}

type returnOnlyBridge struct{}

func (returnOnlyBridge) Middleware() []any { return nil }
func (returnOnlyBridge) Finalize(w stdhttp.ResponseWriter, req any, resp any) any {
	if r, ok := resp.(*http.Response); ok && r != nil {
		return r.Header("X-Fin-Resp", "1")
	}
	return resp
}

func TestFinalizeReturnOnlyKeepsKernelWriteTo(t *testing.T) {
	app := bootBridgeApp(t, returnOnlyBridge{})
	rec := serveOK(t, app)
	if rec.Code != 200 || rec.Body.String() != "handler" {
		t.Fatalf("return-only: status=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Fin-Resp") != "1" {
		t.Fatal("return-only dropped response header")
	}
	if rec.Header().Get("Set-Cookie") == "" {
		t.Fatal("return-only dropped queued cookies")
	}
}

type headerOnlyBridge struct{}

func (headerOnlyBridge) Middleware() []any { return nil }
func (headerOnlyBridge) Finalize(w stdhttp.ResponseWriter, req any, resp any) any {
	w.Header().Set("X-Fin-Direct", "1")
	if r, ok := resp.(*http.Response); ok && r != nil {
		return r.Header("X-Fin-Resp", "1")
	}
	return resp
}

func TestFinalizeHeaderOnlyDoesNotTakeOwnership(t *testing.T) {
	app := bootBridgeApp(t, headerOnlyBridge{})
	rec := serveOK(t, app)
	if rec.Code != 200 || rec.Body.String() != "handler" {
		t.Fatalf("header-only: status=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Fin-Direct") != "1" || rec.Header().Get("X-Fin-Resp") != "1" {
		t.Fatalf("headers=%v", rec.Header())
	}
	if rec.Header().Get("Set-Cookie") == "" {
		t.Fatal("header-only dropped queued cookies")
	}
}

type writeHeaderOnlyBridge struct{}

func (writeHeaderOnlyBridge) Middleware() []any { return nil }
func (writeHeaderOnlyBridge) Finalize(w stdhttp.ResponseWriter, req any, resp any) any {
	w.WriteHeader(stdhttp.StatusAccepted)
	return http.Text("from-resp").Status(stdhttp.StatusNotFound)
}

func TestFinalizeWriteHeaderTakesOwnership(t *testing.T) {
	app := bootBridgeApp(t, writeHeaderOnlyBridge{})
	rec := serveOK(t, app)
	if rec.Code != stdhttp.StatusAccepted {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Body.String() != "" {
		t.Fatalf("kernel WriteTo after WriteHeader: body=%q", rec.Body.String())
	}
}

type writeAndReturnBridge struct{}

func (writeAndReturnBridge) Middleware() []any { return nil }
func (writeAndReturnBridge) Finalize(w stdhttp.ResponseWriter, req any, resp any) any {
	w.Header().Set("X-Fin-Direct", "1")
	w.WriteHeader(stdhttp.StatusAccepted)
	_, _ = io.WriteString(w, "from-w")
	return http.Text("from-resp").Status(stdhttp.StatusNotFound).Header("X-Fin-Resp", "1")
}

func TestFinalizeWriteAndReturnDoesNotMixBody(t *testing.T) {
	app := bootBridgeApp(t, writeAndReturnBridge{})
	rec := serveOK(t, app)
	if rec.Code != stdhttp.StatusAccepted {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Body.String() != "from-w" {
		t.Fatalf("mixed body=%q", rec.Body.String())
	}
	if rec.Header().Get("X-Fin-Resp") == "1" {
		t.Fatal("returned response leaked onto the wire")
	}
	if rec.Header().Get("X-Fin-Direct") != "1" {
		t.Fatal("direct header lost")
	}
}

type writeNilReturnBridge struct{}

func (writeNilReturnBridge) Middleware() []any { return nil }
func (writeNilReturnBridge) Finalize(w stdhttp.ResponseWriter, req any, resp any) any {
	w.WriteHeader(stdhttp.StatusAccepted)
	_, _ = io.WriteString(w, "from-w")
	return nil
}

func TestFinalizeCommitNilReturnKeepsFinalizeBody(t *testing.T) {
	app := bootBridgeApp(t, writeNilReturnBridge{})
	rec := serveOK(t, app)
	if rec.Code != stdhttp.StatusAccepted || rec.Body.String() != "from-w" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

type nilReturnBridge struct{}

func (nilReturnBridge) Middleware() []any { return nil }
func (nilReturnBridge) Finalize(w stdhttp.ResponseWriter, req any, resp any) any {
	return nil
}

func TestFinalizeNilReturnWithoutCommitUsesOriginal(t *testing.T) {
	app := bootBridgeApp(t, nilReturnBridge{})
	rec := serveOK(t, app)
	if rec.Code != 200 || rec.Body.String() != "handler" {
		t.Fatalf("nil return: status=%d body=%q", rec.Code, rec.Body.String())
	}
}
