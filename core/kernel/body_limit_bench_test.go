package kernel

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

func BenchmarkBodyLimitNeedsRouteGET(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if bodyNeedsRoute("GET", 0, 2<<20) {
			b.Fatal("GET consulted the router")
		}
	}
}

func BenchmarkBodyLimitNeedsRouteHEAD(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if bodyNeedsRoute("HEAD", 1<<20, 2<<20) {
			b.Fatal("HEAD consulted the router")
		}
	}
}

func BenchmarkBodyLimitNeedsRouteSmallPOST(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if bodyNeedsRoute("POST", 1024, 2<<20) {
			b.Fatal("small POST consulted the router")
		}
	}
}

func BenchmarkBodyLimitRouteLookup(b *testing.B) {
	app := NewApplication(b.TempDir())
	for i := 0; i < 32; i++ {
		app.router.Get("/s/"+strconv.Itoa(i), func(req *http.Request) *http.Response {
			return http.Text("ok")
		})
	}
	app.router.Post("/webhook", func(req *http.Request) *http.Response {
		return http.Text("ok")
	}).BodyLimit(8 << 20)
	if err := app.router.Freeze(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := app.router.BodyLimitFor("POST", "/webhook"); !ok {
			b.Fatal("missing limit")
		}
	}
}

func BenchmarkHeaderHookGET(b *testing.B) {
	benchHeaderHook(b, "GET /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n")
}
func BenchmarkHeaderHookHEAD(b *testing.B) {
	benchHeaderHook(b, "HEAD /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n")
}
func BenchmarkHeaderHookPOST(b *testing.B) {
	benchHeaderHook(b, "POST /plaintext HTTP/1.1\r\nHost: localhost\r\nContent-Length: 4\r\n\r\nping")
}
func BenchmarkHeaderHookJSON(b *testing.B) {
	benchHeaderHook(b, "POST /plaintext HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 2\r\n\r\n{}")
}
func BenchmarkHeaderHookMultipart(b *testing.B) {
	benchHeaderHook(b, "POST /plaintext HTTP/1.1\r\nHost: localhost\r\nContent-Type: multipart/form-data; boundary=b\r\nContent-Length: 4\r\n\r\n----")
}

func benchHeaderHook(b *testing.B, raw string) {
	app := NewApplication(b.TempDir())
	app.ensureBodyLimits()
	srv := &rawhttp.Server{
		Handler: func(ctx *rawhttp.Ctx) { ctx.SetStatusCode(204) },
		HeaderReceived: func(ctx *rawhttp.Ctx) rawhttp.RequestConfig {
			b.ReportAllocs()
			b.ResetTimer()
			var cfg rawhttp.RequestConfig
			for i := 0; i < b.N; i++ {
				cfg = app.headerBodyConfig(ctx)
			}
			b.StopTimer()
			return cfg
		},
		ReadTimeout:  -1,
		WriteTimeout: -1,
		IdleTimeout:  -1,
	}
	buf := &bytes.Buffer{}
	if err := srv.ServeConn(&ablateMem{r: bytes.NewReader([]byte(raw)), w: buf}); buf.Len() == 0 {
		b.Fatalf("no response: %v", err)
	}
}
