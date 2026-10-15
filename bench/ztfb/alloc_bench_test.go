package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/zatrano/framework/v3/core/kernel"
	khttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

func BenchmarkZatranoPlaintext(b *testing.B) {
	zatranoBench(b, "GET /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n", zatranoRoutesSame)
}

func BenchmarkZatranoJSON(b *testing.B) {
	zatranoBench(b, "GET /json HTTP/1.1\r\nHost: localhost\r\n\r\n", zatranoRoutesSame)
}

func BenchmarkZatranoUsers(b *testing.B) {
	zatranoBench(b, "GET /users/42 HTTP/1.1\r\nHost: localhost\r\n\r\n", zatranoRoutesSame)
}

func BenchmarkZatranoEcho(b *testing.B) {
	zatranoBench(b, "POST /echo HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 27\r\n\r\n{\"message\":\"Hello, World!\"}", zatranoRoutesSame)
}

func BenchmarkZatranoJSONFramework(b *testing.B) {
	zatranoBench(b, "GET /json HTTP/1.1\r\nHost: localhost\r\n\r\n", zatranoRoutesFramework)
}

func BenchmarkZatranoUsersFramework(b *testing.B) {
	zatranoBench(b, "GET /users/42 HTTP/1.1\r\nHost: localhost\r\n\r\n", zatranoRoutesFramework)
}

func BenchmarkFiberPlaintext(b *testing.B) {
	fiberBench(b, "GET /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n", fiberRoutesSame)
}

func BenchmarkFiberJSON(b *testing.B) {
	fiberBench(b, "GET /json HTTP/1.1\r\nHost: localhost\r\n\r\n", fiberRoutesSame)
}

func BenchmarkFiberUsers(b *testing.B) {
	fiberBench(b, "GET /users/42 HTTP/1.1\r\nHost: localhost\r\n\r\n", fiberRoutesSame)
}

func BenchmarkFiberEcho(b *testing.B) {
	fiberBench(b, "POST /echo HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 27\r\n\r\n{\"message\":\"Hello, World!\"}", fiberRoutesSame)
}

func BenchmarkFiberJSONFramework(b *testing.B) {
	fiberBench(b, "GET /json HTTP/1.1\r\nHost: localhost\r\n\r\n", fiberRoutesFramework)
}

func BenchmarkFiberUsersFramework(b *testing.B) {
	fiberBench(b, "GET /users/42 HTTP/1.1\r\nHost: localhost\r\n\r\n", fiberRoutesFramework)
}

func BenchmarkFiberTunedPlaintext(b *testing.B) {
	fiberBenchApp(b, "GET /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n", newFiberTunedApp())
}

func BenchmarkFiberTunedJSON(b *testing.B) {
	fiberBenchApp(b, "GET /json HTTP/1.1\r\nHost: localhost\r\n\r\n", newFiberTunedApp())
}

func BenchmarkFiberTunedUsers(b *testing.B) {
	fiberBenchApp(b, "GET /users/42 HTTP/1.1\r\nHost: localhost\r\n\r\n", newFiberTunedApp())
}

func BenchmarkFiberTunedEcho(b *testing.B) {
	fiberBenchApp(b, "POST /echo HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 27\r\n\r\n{\"message\":\"Hello, World!\"}", newFiberTunedApp())
}

func BenchmarkFiberTunedJSONFramework(b *testing.B) {
	fiberBenchApp(b, "GET /json HTTP/1.1\r\nHost: localhost\r\n\r\n", newFiberTuned(func(app *fiber.App) {
		app.Get("/json", func(c fiber.Ctx) error {
			return c.JSON(fiberJSONPayload, jsonType)
		})
	}))
}

func BenchmarkFiberTunedUsersFramework(b *testing.B) {
	fiberBenchApp(b, "GET /users/42 HTTP/1.1\r\nHost: localhost\r\n\r\n", newFiberTuned(func(app *fiber.App) {
		app.Get("/users/:id", func(c fiber.Ctx) error {
			fiberUserPayload["id"] = c.Params("id")
			return c.JSON(fiberUserPayload, jsonType)
		})
	}))
}

var (
	fiberJSONPayload   = map[string]string{"message": "Hello, World!"}
	fiberUserPayload   = map[string]string{"id": ""}
	zatranoJSONPayload = map[string]string{"message": "Hello, World!"}
	zatranoUserPayload = map[string]string{"id": ""}
)

func zatranoRoutesSame(app *kernel.Application) {
	app.Router().Get("/plaintext", func(*khttp.Request) *khttp.Response {
		return khttp.Text(helloBody)
	})
	app.Router().Get("/json", func(*khttp.Request) *khttp.Response {
		return khttp.JSON(map[string]string{"message": "Hello, World!"})
	})
	app.Router().Get("/users/{id}", func(req *khttp.Request) *khttp.Response {
		return khttp.JSON(map[string]string{"id": req.Route("id")})
	})
	app.Router().Post("/echo", func(req *khttp.Request) *khttp.Response {
		var m map[string]any
		if err := req.BindJSON(&m); err != nil {
			return khttp.BadRequest("bad json")
		}
		return khttp.JSON(m)
	})
}

func zatranoRoutesFramework(app *kernel.Application) {
	app.Router().Get("/json", func(*khttp.Request) *khttp.Response {
		return khttp.JSON(zatranoJSONPayload)
	})
	app.Router().Get("/users/{id}", func(req *khttp.Request) *khttp.Response {
		zatranoUserPayload["id"] = req.Route("id")
		return khttp.JSON(zatranoUserPayload)
	})
}

func zatranoBench(b *testing.B, raw string, routes func(*kernel.Application)) {
	b.Helper()
	b.Setenv("APP_ENV", "production")
	b.Setenv("APP_DEBUG", "false")
	b.Setenv("LOG_LEVEL", "error")
	b.Setenv("APP_KEY", "zatrano-ci-test-key-not-prod!!!!")
	b.Setenv("HTTP_POOL_REQUESTS", "")
	dir := b.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		b.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	b.Cleanup(func() {
		if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
			_ = c.Close()
		}
	})
	routes(app)
	if err := app.Bootstrap(); err != nil {
		b.Fatal(err)
	}
	srv := &rawhttp.Server{
		Handler:           func(ctx *rawhttp.Ctx) { app.Handle(ctx) },
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}
	pipeBench(b, raw, func(ln net.Listener) error { return srv.Serve(ln) })
}

func fiberRoutesSame(app *fiber.App) {
	app.Get("/plaintext", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; charset=utf-8")
		return c.SendString(helloBody)
	})
	app.Get("/json", func(c fiber.Ctx) error {
		return c.JSON(map[string]string{"message": "Hello, World!"})
	})
	app.Get("/users/:id", func(c fiber.Ctx) error {
		return c.JSON(map[string]string{"id": c.Params("id")})
	})
	app.Post("/echo", func(c fiber.Ctx) error {
		var m map[string]any
		if err := codecUnmarshal(c.Body(), &m); err != nil {
			return c.Status(400).SendString("bad json")
		}
		return c.JSON(m)
	})
}

func fiberRoutesFramework(app *fiber.App) {
	app.Get("/json", func(c fiber.Ctx) error {
		return c.JSON(fiberJSONPayload)
	})
	app.Get("/users/:id", func(c fiber.Ctx) error {
		fiberUserPayload["id"] = c.Params("id")
		return c.JSON(fiberUserPayload)
	})
}

func fiberBench(b *testing.B, raw string, routes func(*fiber.App)) {
	b.Helper()
	app := fiber.New(fiber.Config{
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
		JSONEncoder:  codecMarshal,
		JSONDecoder:  codecUnmarshal,
	})
	app.Use(recover.New())
	app.Use(requestid.New(requestid.Config{
		Header: "X-Request-Id",
		Generator: func() string {
			var buf [16]byte
			if _, err := rand.Read(buf[:]); err != nil {
				return "00000000000000000000000000000000"
			}
			return hex.EncodeToString(buf[:])
		},
	}))
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Frame-Options", "SAMEORIGIN")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		return c.Next()
	})
	routes(app)
	pipeBench(b, raw, func(ln net.Listener) error {
		return app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true})
	})
}

func fiberBenchApp(b *testing.B, raw string, app *fiber.App) {
	b.Helper()
	pipeBench(b, raw, func(ln net.Listener) error {
		defer func() { _ = app.Shutdown() }()
		return app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true})
	})
}

func pipeBench(b *testing.B, raw string, serve func(net.Listener) error) {
	b.Helper()
	left, right := net.Pipe()
	ln := &oneConnListener{conn: left, stop: make(chan struct{})}
	errCh := make(chan error, 1)
	go func() { errCh <- serve(ln) }()
	b.Cleanup(func() {
		ln.Close()
		_ = left.Close()
		_ = right.Close()
	})

	req := []byte(raw)
	ack := make([]byte, 2048)
	for i := 0; i < 64; i++ {
		if _, err := right.Write(req); err != nil {
			b.Fatal(err)
		}
		if err := drainOneResponse(right, ack); err != nil {
			b.Fatal(err)
		}
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := right.Write(req); err != nil {
			b.Fatal(err)
		}
		if err := drainOneResponse(right, ack); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	runtime.ReadMemStats(&after)
	if b.N > 0 {
		b.ReportMetric(float64(after.Mallocs-before.Mallocs)/float64(b.N), "mallocs/req")
		b.ReportMetric(float64(after.TotalAlloc-before.TotalAlloc)/float64(b.N), "bytes/req")
	}
	select {
	case err := <-errCh:
		if err != nil {
			b.Log(err)
		}
	default:
	}
}

type oneConnListener struct {
	conn net.Conn
	once sync.Once
	stop chan struct{}
}

func (l *oneConnListener) Accept() (net.Conn, error) {
	var conn net.Conn
	l.once.Do(func() { conn = l.conn })
	if conn != nil {
		return conn, nil
	}
	<-l.stop
	return nil, io.EOF
}

func (l *oneConnListener) Close() error {
	select {
	case <-l.stop:
	default:
		close(l.stop)
	}
	return nil
}

func (l *oneConnListener) Addr() net.Addr { return pipeAddr{} }

type pipeAddr struct{}

func (pipeAddr) Network() string { return "tcp" }
func (pipeAddr) String() string  { return "127.0.0.1:1" }

func drainOneResponse(c net.Conn, buf []byte) error {
	n := 0
	for {
		if n == len(buf) {
			return io.ErrShortBuffer
		}
		r, err := c.Read(buf[n:])
		n += r
		if err != nil {
			return err
		}
		if i := bytes.Index(buf[:n], headerSplit); i >= 0 {
			body := n - (i + 4)
			need := contentLen(buf[:i])
			for body < need {
				if n == len(buf) {
					return io.ErrShortBuffer
				}
				r, err = c.Read(buf[n:])
				n += r
				body += r
				if err != nil {
					return err
				}
			}
			return nil
		}
	}
}

var headerSplit = []byte("\r\n\r\n")
var contentLenKey = []byte("Content-Length: ")

func contentLen(header []byte) int {
	i := bytes.Index(header, contentLenKey)
	if i < 0 {
		return 0
	}
	rest := header[i+len(contentLenKey):]
	n := 0
	for _, c := range rest {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}
