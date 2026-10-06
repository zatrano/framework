package kernel

import (
	"bytes"
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/middleware"
	"github.com/zatrano/rawhttp"
)

type bodyLimitHit struct {
	method string
	path   string
	limit  int64
	set    bool
}

func TestBodyLimitMatchesDispatchCorpus(t *testing.T) {
	app := NewApplication(t.TempDir())
	var saw bodyLimitHit
	reg := func(method, path string, limit int64, set bool) {
		t.Helper()
		handler := func(*http.Request) *http.Response {
			saw = bodyLimitHit{method: method, path: path, limit: limit, set: set}
			return http.Text("ok")
		}
		switch method {
		case "POST":
			rt := app.router.Post(path, handler)
			if set {
				rt.BodyLimit(limit)
			}
		case "PUT":
			rt := app.router.Put(path, handler)
			if set {
				rt.BodyLimit(limit)
			}
		case "PATCH":
			rt := app.router.Patch(path, handler)
			if set {
				rt.BodyLimit(limit)
			}
		case "DELETE":
			rt := app.router.Delete(path, handler)
			if set {
				rt.BodyLimit(limit)
			}
		default:
			t.Fatalf("method %s", method)
		}
	}
	reg("POST", "/hook", 8<<20, true)
	reg("PUT", "/hook", 8<<20, true)
	reg("POST", "/x", 4<<20, true)
	reg("PUT", "/items/{id}", 6<<20, true)
	reg("PATCH", "/x", 100, true)
	reg("DELETE", "/x", -1, true)
	reg("POST", "/only", 0, false)
	if err := app.router.Freeze(); err != nil {
		t.Fatal(err)
	}

	corpus := []struct {
		method   string
		path     string
		ct       string
		override string
		body     string
		cl       int
	}{
		{"POST", "/hook", "application/json", "", "", 3 << 20},
		{"POST", "/hook/", "application/json", "", "", 3 << 20},
		{"PUT", "/hook", "application/json", "", "", 3 << 20},
		{"PUT", "/hook/", "application/json", "", "", 3 << 20},
		{"POST", "/hook", "application/json", "PUT", "", 3 << 20},
		{"POST", "/hook/", "application/json", "PUT", "", 3 << 20},
		{"POST", "/hook", "application/json", "PATCH", "", 3 << 20},
		{"POST", "/hook", "application/json", "DELETE", "", 3 << 20},
		{"POST", "/hook", "application/json", "GET", "", 3 << 20},
		{"POST", "/Hook", "application/json", "", "", 3 << 20},
		{"POST", "/HOOK", "application/json", "PUT", "", 3 << 20},
		{"POST", "/%68ook", "application/json", "", "", 3 << 20},
		{"POST", "/hook/.", "application/json", "", "", 3 << 20},
		{"POST", "/./hook", "application/json", "", "", 3 << 20},
		{"POST", "/hook/./", "application/json", "", "", 3 << 20},
		{"POST", "/a//hook", "application/json", "", "", 3 << 20},
		{"POST", "//hook", "application/json", "", "", 3 << 20},
		{"POST", "/x", "application/json", "", "", 3 << 20},
		{"POST", "/x/", "application/json", "", "", 3 << 20},
		{"POST", "/X", "application/json", "", "", 3 << 20},
		{"POST", "/%78", "application/json", "", "", 3 << 20},
		{"POST", "/x/./", "application/json", "", "", 3 << 20},
		{"PATCH", "/x", "application/json", "", "", 3 << 20},
		{"PATCH", "/x/", "text/plain", "", "", 3 << 20},
		{"DELETE", "/x", "application/octet-stream", "", "", 40 << 20},
		{"DELETE", "/x/", "application/octet-stream", "", "", 40 << 20},
		{"POST", "/x", "application/json", "DELETE", "", 3 << 20},
		{"POST", "/x/", "application/json", "PATCH", "", 3 << 20},
		{"PUT", "/items/7", "application/json", "", "", 3 << 20},
		{"PUT", "/items/7/", "application/json", "", "", 3 << 20},
		{"PUT", "/items//7", "application/json", "", "", 3 << 20},
		{"POST", "/items/7", "application/json", "PUT", "", 3 << 20},
		{"POST", "/only", "application/json", "", "", 3 << 20},
		{"POST", "/only/", "text/plain", "", "", 3 << 20},
		{"POST", "/missing", "application/json", "", "", 3 << 20},
		{"GET", "/hook", "application/json", "", "", 3 << 20},
		{"HEAD", "/hook", "application/json", "", "", 0},
		{"POST", "/hook", "application/x-www-form-urlencoded", "", "_method=PUT", 3 << 20},
		{"POST", "/hook", "application/x-www-form-urlencoded", "", "", 3 << 20},
		{"POST", "/x", "application/x-www-form-urlencoded", "", "_method=PATCH", 3 << 20},
		{"POST", "/x", "multipart/form-data; boundary=b", "", "", 3 << 20},
		{"POST", "/hook", "application/json", "put", "", 3 << 20},
		{"POST", "/x", "application/vnd.api+json", "", "", 3 << 20},
		{"POST", "/x/", "application/json; charset=utf-8", "", "", 1 << 20},
	}
	if len(corpus) < 40 {
		t.Fatalf("corpus %d, want at least 40", len(corpus))
	}
	for _, tc := range corpus {
		saw = bodyLimitHit{}
		raw := httptest.NewRequest(tc.method, "http://localhost"+tc.path, strings.NewReader(tc.body))
		if tc.ct != "" {
			raw.Header.Set("Content-Type", tc.ct)
		}
		if tc.override != "" {
			raw.Header.Set("X-HTTP-Method-Override", tc.override)
		}
		req := http.RequestFromHTTP(raw)
		middleware.ApplyMethodOverride(req)
		resp := app.router.Dispatch(req)
		dispatchCap := http.HeaderBodyLimit(tc.ct)
		if resp != nil && resp.StatusCode() != 404 && saw.path != "" {
			dispatchCap = routeBodyCap(saw.limit)
			if !saw.set {
				dispatchCap = http.HeaderBodyLimit(tc.ct)
			}
		}
		cfg := app.decideBodyLimit(tc.method, tc.path, tc.ct, tc.override, tc.cl)
		granted := int64(cfg.MaxRequestBodySize)
		if granted <= 0 {
			granted = http.MaxRequestBytes()
		}
		if granted > dispatchCap {
			t.Fatalf("%s %s override=%q ct=%q header cap %d exceeds dispatch cap %d (route %s %s)",
				tc.method, tc.path, tc.override, tc.ct, granted, dispatchCap, saw.method, saw.path)
		}
	}
}

func TestBodyLimitOverrideAndPathVariants(t *testing.T) {
	app := NewApplication(t.TempDir())
	app.router.Put("/hook", func(*http.Request) *http.Response {
		return http.Text("ok")
	}).BodyLimit(8 << 20)
	app.router.Post("/x", func(*http.Request) *http.Response {
		return http.Text("ok")
	}).BodyLimit(4 << 20)
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var hit atomic.Int32
	srv.Handler = func(ctx *rawhttp.Ctx) {
		hit.Add(1)
		ctx.SetStatusCode(200)
		ctx.SetBodyString("ok")
	}
	ln := serveTestServer(t, srv)
	addr := ln.Addr().String()

	status, body := postOverride(t, addr, "/hook", "PUT", "application/json", bytes.Repeat([]byte("j"), 3<<20))
	if status != 200 || body != "ok" || hit.Load() != 1 {
		t.Fatalf("POST+override PUT 3 MiB status=%d body=%q hit=%d", status, body, hit.Load())
	}
	status, body = postRawPath(t, addr, "/x/", "application/json", bytes.Repeat([]byte("j"), 3<<20))
	if status != 200 || body != "ok" || hit.Load() != 2 {
		t.Fatalf("/x/ status=%d body=%q hit=%d", status, body, hit.Load())
	}
	before := hit.Load()
	if status = postDeclaredPath(t, addr, "/X", "application/json", 3<<20); status != 413 || hit.Load() != before {
		t.Fatalf("/X status=%d hit=%d", status, hit.Load())
	}
	if status = postDeclaredPath(t, addr, "/%78", "application/json", 3<<20); status != 413 || hit.Load() != before {
		t.Fatalf("/%%78 status=%d hit=%d", status, hit.Load())
	}
	if status = postDeclaredPath(t, addr, "//x", "application/json", 3<<20); status == 200 || hit.Load() != before {
		t.Fatalf("//x status=%d hit=%d", status, hit.Load())
	}
}

func TestBodyLimitAboveCeiling(t *testing.T) {
	app := NewApplication(t.TempDir())
	app.router.Post("/wide", func(*http.Request) *http.Response {
		return http.Text("ok")
	}).BodyLimit(64 << 20)
	app.router.Post("/unsustainable", func(*http.Request) *http.Response {
		return http.Text("no")
	}).BodyLimit(300 << 20)
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if srv.MaxRequestBodySize > 32<<20 && srv.MaxRequestBodySize < 64<<20 {
		t.Fatalf("ceiling %d is not the 32 MiB default this test assumes", srv.MaxRequestBodySize)
	}
	var hit atomic.Int32
	srv.Handler = func(ctx *rawhttp.Ctx) {
		hit.Add(1)
		ctx.SetStatusCode(200)
		ctx.SetBodyString("ok")
	}
	// A known length above the read buffer must still be accepted when the
	// route cap is above the server ceiling. v0.2.2 answers 431 before the cap
	// unless the buffer can hold the body.
	srv.ReadBufferSize = 40 << 20
	ln := serveTestServer(t, srv)
	addr := ln.Addr().String()

	cfg := app.decideBodyLimit("POST", "/wide", "application/octet-stream", "", 33<<20)
	if int64(cfg.MaxRequestBodySize) <= http.MaxRequestBytes() {
		t.Fatalf("RequestConfig did not exceed the server ceiling: %d", cfg.MaxRequestBodySize)
	}
	status, body := postRawPath(t, addr, "/wide", "application/octet-stream", bytes.Repeat([]byte("o"), 33<<20))
	if status != 200 || body != "ok" || hit.Load() != 1 {
		t.Fatalf("33 MiB above ceiling status=%d body=%q hit=%d", status, body, hit.Load())
	}
	before := hit.Load()
	status = postDeclaredPath(t, addr, "/unsustainable", "application/json", 3<<20)
	if status != 413 || hit.Load() != before {
		t.Fatalf("cap above budget status=%d hit=%d, want 413", status, hit.Load())
	}
}

func postOverride(t *testing.T, addr, path, override, contentType string, body []byte) (int, string) {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var buf bytes.Buffer
	buf.WriteString("POST " + path + " HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n")
	buf.WriteString("X-HTTP-Method-Override: " + override + "\r\n")
	if contentType != "" {
		buf.WriteString("Content-Type: " + contentType + "\r\n")
	}
	buf.WriteString("Content-Length: " + itoa(len(body)) + "\r\n\r\n")
	buf.Write(body)
	if _, err := c.Write(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(c)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	return statusCode(text), lastBody(text)
}

func lastBody(text string) string {
	if i := strings.Index(text, "\r\n\r\n"); i >= 0 {
		return text[i+4:]
	}
	return ""
}
