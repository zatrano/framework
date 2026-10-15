package kernel_test

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel"
	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

func buildRawRequest(method, path string, headers map[string]string, body string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s HTTP/1.1\r\n", method, path)
	b.WriteString("Host: localhost\r\n")
	b.WriteString("Connection: close\r\n")
	for k, v := range headers {
		fmt.Fprintf(&b, "%s: %s\r\n", k, v)
	}
	if body != "" || method == "POST" || method == "PUT" || method == "PATCH" {
		fmt.Fprintf(&b, "Content-Length: %d\r\n", len(body))
	}
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}

func serveHandle(t *testing.T, app *kernel.Application, method, path string, headers map[string]string, body string) http.ExchangeResult {
	t.Helper()
	raw := buildRawRequest(method, path, headers, body)
	er, err := http.ExchangeForTest(func(ctx *rawhttp.Ctx) { app.Handle(ctx) }, raw)
	if err != nil {
		t.Fatalf("ExchangeForTest: %v (raw=%q)", err, er.Raw)
	}
	return er
}

func bootPublicApp(t *testing.T) *kernel.Application {
	t.Helper()
	dir := t.TempDir()
	public := filepath.Join(dir, "public")
	if err := os.MkdirAll(public, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(public, "app.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	t.Cleanup(func() { closeAppLog(t, app) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	return app
}

func bootProductionPublicApp(t testing.TB, routes func(*kernel.Application)) (*kernel.Application, string) {
	t.Helper()
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	dir := t.TempDir()
	public := filepath.Join(dir, "public")
	if err := os.MkdirAll(filepath.Join(public, "css"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(public, "app.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(public, "css", "app.css"), []byte("nested{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	t.Cleanup(func() { closeAppLog(t, app) })
	if routes != nil {
		routes(app)
	}
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	return app, dir
}

func TestPlaintextCarriesGeneratedRequestID(t *testing.T) {
	app, _ := bootProductionPublicApp(t, benchRoutes)
	er := serveHandle(t, app, "GET", "/plaintext", nil, "")
	if er.Status != 200 || string(er.Body) != "Hello, World!" {
		t.Fatalf("status=%d body=%q", er.Status, er.Body)
	}
	id := er.Header.Get("X-Request-Id")
	if len(id) != 32 {
		t.Fatalf("request id=%q", id)
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Fatalf("request id=%q", id)
		}
	}
}

// benchRoutes is the measured server: the same handlers as the Fiber pair.
func benchRoutes(app *kernel.Application) {
	app.Router().Get("/plaintext", func(*http.Request) *http.Response {
		return http.Text("Hello, World!")
	})
	app.Router().Get("/json", func(*http.Request) *http.Response {
		return http.JSON(map[string]string{"message": "Hello, World!"})
	})
	app.Router().Get("/users/{id}", func(req *http.Request) *http.Response {
		return http.JSON(map[string]string{"id": req.Route("id")})
	})
	app.Router().Post("/echo", func(req *http.Request) *http.Response {
		var m map[string]any
		if err := req.BindJSON(&m); err != nil {
			return http.BadRequest("bad json")
		}
		return http.JSON(m)
	})
}

// benchRoutesFramework is the same handlers with the map literal hoisted,
// so the remaining allocs are the framework path.
var (
	benchJSONPayload = map[string]string{"message": "Hello, World!"}
	benchUserPayload = map[string]string{"id": ""}
)

func benchRoutesFramework(app *kernel.Application) {
	app.Router().Get("/plaintext", func(*http.Request) *http.Response {
		return http.Text("Hello, World!")
	})
	app.Router().Get("/json", func(*http.Request) *http.Response {
		return http.JSON(benchJSONPayload)
	})
	app.Router().Get("/users/{id}", func(req *http.Request) *http.Response {
		benchUserPayload["id"] = req.Route("id")
		return http.JSON(benchUserPayload)
	})
	app.Router().Post("/echo", func(req *http.Request) *http.Response {
		var m map[string]any
		if err := req.BindJSON(&m); err != nil {
			return http.BadRequest("bad json")
		}
		return http.JSON(m)
	})
}

func BenchmarkHandlePlaintext(b *testing.B) {
	benchHandle(b, "GET /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n")
}

func BenchmarkHandleJSON(b *testing.B) {
	benchHandle(b, "GET /json HTTP/1.1\r\nHost: localhost\r\n\r\n")
}

func BenchmarkHandleUsers(b *testing.B) {
	benchHandle(b, "GET /users/42 HTTP/1.1\r\nHost: localhost\r\n\r\n")
}

func BenchmarkHandleEcho(b *testing.B) {
	benchHandle(b, "POST /echo HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 27\r\n\r\n{\"message\":\"Hello, World!\"}")
}

func BenchmarkHandleJSONFramework(b *testing.B) {
	benchHandleRoutes(b, "GET /json HTTP/1.1\r\nHost: localhost\r\n\r\n", benchRoutesFramework)
}

func BenchmarkHandleUsersFramework(b *testing.B) {
	benchHandleRoutes(b, "GET /users/42 HTTP/1.1\r\nHost: localhost\r\n\r\n", benchRoutesFramework)
}

// benchHandle pipelines one keep-alive connection. -benchmem is the process
// malloc delta. The same window is also reported as server MemStats per
// request after a warmup, so the harness writes are visible separately.
func bootBenchAppRoutes(b *testing.B, routes func(*kernel.Application)) *kernel.Application {
	b.Helper()
	b.Setenv("APP_ENV", "production")
	b.Setenv("APP_DEBUG", "false")
	b.Setenv("APP_KEY", strings.Repeat("s", 32))
	dir := b.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		b.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	b.Cleanup(func() { closeAppLog(b, app) })
	routes(app)
	if err := app.Bootstrap(); err != nil {
		b.Fatal(err)
	}
	return app
}

func benchHandle(b *testing.B, raw string) {
	benchHandleRoutes(b, raw, benchRoutes)
}

func benchHandleRoutes(b *testing.B, raw string, routes func(*kernel.Application)) {
	b.Helper()
	app := bootBenchAppRoutes(b, routes)
	srv := &rawhttp.Server{Handler: func(ctx *rawhttp.Ctx) { app.Handle(ctx) }}
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	go func() { _ = srv.ServeConn(left) }()

	req := []byte(raw)
	ack := make([]byte, 1024)
	// Warm the pools and the keep-alive header cache.
	for i := 0; i < 64; i++ {
		if _, err := right.Write(req); err != nil {
			b.Fatal(err)
		}
		if err := drainResponse(right, ack); err != nil {
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
		if err := drainResponse(right, ack); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	runtime.ReadMemStats(&after)
	if b.N > 0 {
		b.ReportMetric(float64(after.Mallocs-before.Mallocs)/float64(b.N), "mallocs/req")
		b.ReportMetric(float64(after.TotalAlloc-before.TotalAlloc)/float64(b.N), "bytes/req")
		b.ReportMetric(float64(after.NumGC-before.NumGC), "gc/window")
	}
}

func drainResponse(c net.Conn, buf []byte) error {
	// Read until the header blank line, then Content-Length bytes.
	// The buffer is reused. A short read fills it; responses here are < 512 bytes.
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
		if i := bytes.Index(buf[:n], headerEnd); i >= 0 {
			header := buf[:i]
			body := n - (i + 4)
			need := contentLength(header)
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

var (
	headerEnd = []byte("\r\n\r\n")
	clKey     = []byte("Content-Length: ")
)

func contentLength(header []byte) int {
	i := bytes.Index(header, clKey)
	if i < 0 {
		return 0
	}
	rest := header[i+len(clKey):]
	n := 0
	for _, c := range rest {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}
