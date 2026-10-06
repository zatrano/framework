package kernel

import (
	"bytes"
	"testing"

	khttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

func TestHeaderHookAllocs(t *testing.T) {
	app := NewApplication(t.TempDir())
	app.ensureBodyLimits()
	// Bodyless requests never classify the media type or copy the path.
	for _, raw := range []string{
		"GET /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n",
		"HEAD /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n",
		"OPTIONS /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n",
		"DELETE /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n",
		"POST /plaintext HTTP/1.1\r\nHost: localhost\r\nContent-Length: 0\r\n\r\n",
	} {
		n, cfg := measureHook(t, app, raw)
		if n != 0 || cfg.MaxRequestBodySize != 0 {
			t.Fatalf("%q allocs=%v cfg=%d", raw[:4], n, cfg.MaxRequestBodySize)
		}
	}

	// Small JSON is under the content-type cap and this router has no tighter
	// BodyLimit, so the hook does not copy the path. The one-alloc budget is
	// the path string BodyLimitFor needs when a lookup actually runs.
	n, cfg := measureHook(t, app, "POST /plaintext HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 2\r\n\r\n{}")
	if n > 1 {
		t.Fatalf("small JSON allocs=%v", n)
	}
	if cfg.MaxRequestBodySize != 2<<20 {
		t.Fatalf("json cap=%d", cfg.MaxRequestBodySize)
	}

	n, cfg = measureHook(t, app, "POST /plaintext HTTP/1.1\r\nHost: localhost\r\nContent-Type: multipart/form-data; boundary=b\r\nContent-Length: 4\r\n\r\n----")
	if n > 1 {
		t.Fatalf("multipart allocs=%v", n)
	}
	if cfg.MaxRequestBodySize != 32<<20 {
		t.Fatalf("multipart cap=%d", cfg.MaxRequestBodySize)
	}

	app.router.Post("/hook", func(*khttp.Request) *khttp.Response {
		return khttp.Text("ok")
	}).BodyLimit(4096)
	if err := app.router.Freeze(); err != nil {
		t.Fatal(err)
	}
	n, cfg = measureHook(t, app, "POST /hook HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 2\r\n\r\n{}")
	if n > 1 {
		t.Fatalf("lookup allocs=%v want <= 1 (path string)", n)
	}
	if cfg.MaxRequestBodySize != 4096 {
		t.Fatalf("hook cap=%d", cfg.MaxRequestBodySize)
	}
}

func measureHook(t *testing.T, app *Application, raw string) (float64, rawhttp.RequestConfig) {
	t.Helper()
	var n float64
	var cfg rawhttp.RequestConfig
	srv := &rawhttp.Server{
		Handler: func(ctx *rawhttp.Ctx) { ctx.SetStatusCode(204) },
		HeaderReceived: func(ctx *rawhttp.Ctx) rawhttp.RequestConfig {
			n = testing.AllocsPerRun(200, func() {
				cfg = app.headerBodyConfig(ctx)
			})
			return cfg
		},
		ReadTimeout:  -1,
		WriteTimeout: -1,
		IdleTimeout:  -1,
	}
	buf := &bytes.Buffer{}
	err := srv.ServeConn(&ablateMem{r: bytes.NewReader([]byte(raw)), w: buf})
	if buf.Len() == 0 {
		t.Fatalf("no response: %v", err)
	}
	return n, cfg
}
