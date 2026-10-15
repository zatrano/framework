package main

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"os"
	"os/exec"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// TestFiberPlaintextAllocObjects profiles the official Fiber plaintext path
// (requestid middleware, its generator, recover, and the four c.Set headers).
// The tuned server is not this profile.
func TestFiberPlaintextAllocObjects(t *testing.T) {
	runtime.MemProfileRate = 1
	t.Cleanup(func() { runtime.MemProfileRate = 512 * 1024 })
	app := fiberOfficialApp(t, fiberRoutesSame)
	_, right := pipePair(t, app)
	req := []byte("GET /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n")
	ack := make([]byte, 2048)
	for i := 0; i < 64; i++ {
		if _, err := right.Write(req); err != nil {
			t.Fatal(err)
		}
		if err := drainOneResponse(right, ack); err != nil {
			t.Fatal(err)
		}
	}

	runtime.MemProfileRate = 1
	t.Cleanup(func() { runtime.MemProfileRate = 512 * 1024 })
	dir := t.TempDir()
	before := dir + "/before.pprof"
	after := dir + "/after.pprof"
	runtime.GC()
	runtime.GC()
	writeAllocProfile(t, before)

	const n = 2000
	var m0, m1 runtime.MemStats
	runtime.ReadMemStats(&m0)
	for i := 0; i < n; i++ {
		if _, err := right.Write(req); err != nil {
			t.Fatal(err)
		}
		if err := drainOneResponse(right, ack); err != nil {
			t.Fatal(err)
		}
	}
	runtime.ReadMemStats(&m1)
	runtime.GC()
	runtime.GC()
	writeAllocProfile(t, after)
	t.Logf("mallocs/req %.3f bytes/req %.1f", float64(m1.Mallocs-m0.Mallocs)/n, float64(m1.TotalAlloc-m0.TotalAlloc)/n)

	cmd := exec.Command("go", "tool", "pprof", "-top", "-sample_index=alloc_objects", "-nodecount=20", "-base", before, after)
	out, err := cmd.CombinedOutput()
	t.Logf("pprof alloc_objects diff\n%s", out)
	if err != nil {
		t.Fatalf("pprof: %v", err)
	}
	listed := exec.Command("go", "tool", "pprof", "-list", "requestid.New.func1", "-sample_index=alloc_objects", "-base", before, after)
	listOut, listErr := listed.CombinedOutput()
	t.Logf("pprof list requestid\n%s", listOut)
	if listErr != nil {
		t.Logf("pprof list: %v", listErr)
	}
}

func fiberOfficialApp(t *testing.T, routes func(*fiber.App)) *fiber.App {
	t.Helper()
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
	return app
}

func pipePair(t *testing.T, app *fiber.App) (net.Conn, net.Conn) {
	t.Helper()
	left, right := net.Pipe()
	ln := &oneConnListener{conn: left, stop: make(chan struct{})}
	go func() { _ = app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()
	t.Cleanup(func() {
		ln.Close()
		_ = app.Shutdown()
		_ = left.Close()
		_ = right.Close()
	})
	return left, right
}

func writeAllocProfile(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := pprof.Lookup("allocs").WriteTo(f, 0); err != nil {
		t.Fatal(err)
	}
}
