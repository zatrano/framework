package main

import (
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/zatrano/framework/v3/core/kernel"
	"github.com/zatrano/rawhttp"
)

// TestPipeAllocSplitPlaintext runs ZATRANO, official Fiber, and tuned Fiber
// through one keep-alive net.Pipe and the same client. Stacks that contain
// pipeDeadline are the pipe. The rest are the server.
func TestPipeAllocSplitPlaintext(t *testing.T) {
	const raw = "GET /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n"
	cases := []struct {
		name  string
		serve func(*testing.T) func(net.Listener) error
	}{
		{"zatrano", zatranoPipeServe},
		{"fiber", fiberPipeServe},
		{"fiber-tuned", tunedPipeServe},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			total, pipe := pipeAllocs(t, raw, tc.serve(t))
			t.Logf("%s total %.3f pipe %.3f server %.3f alloc/req", tc.name, total, pipe, total-pipe)
		})
	}
}

func zatranoPipeServe(t *testing.T) func(net.Listener) error {
	t.Helper()
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("APP_KEY", "zatrano-ci-test-key-not-prod!!!!")
	t.Setenv("HTTP_POOL_REQUESTS", "")
	dir := t.TempDir()
	if err := os.MkdirAll(dir+"/public", 0o755); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	t.Cleanup(func() {
		if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
			_ = c.Close()
		}
	})
	zatranoRoutesSame(app)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	srv := &rawhttp.Server{
		Handler:           func(ctx *rawhttp.Ctx) { app.Handle(ctx) },
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return func(ln net.Listener) error { return srv.Serve(ln) }
}

func fiberPipeServe(t *testing.T) func(net.Listener) error {
	t.Helper()
	app := fiberOfficialApp(t, fiberRoutesSame)
	return func(ln net.Listener) error {
		defer func() { _ = app.Shutdown() }()
		return app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true})
	}
}

func tunedPipeServe(t *testing.T) func(net.Listener) error {
	t.Helper()
	app := newFiberTunedApp()
	return func(ln net.Listener) error {
		defer func() { _ = app.Shutdown() }()
		return app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true})
	}
}

func pipeAllocs(t *testing.T, raw string, serve func(net.Listener) error) (total, pipe float64) {
	t.Helper()
	runtime.MemProfileRate = 1
	t.Cleanup(func() { runtime.MemProfileRate = 512 * 1024 })

	left, right := net.Pipe()
	ln := &oneConnListener{conn: left, stop: make(chan struct{})}
	go func() { _ = serve(ln) }()
	t.Cleanup(func() {
		ln.Close()
		_ = left.Close()
		_ = right.Close()
	})

	req := []byte(raw)
	ack := make([]byte, 2048)
	for i := 0; i < 64; i++ {
		if _, err := right.Write(req); err != nil {
			t.Fatal(err)
		}
		if err := drainOneResponse(right, ack); err != nil {
			t.Fatal(err)
		}
	}
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
	total = float64(m1.Mallocs-m0.Mallocs) / float64(n)

	cmd := exec.Command("go", "tool", "pprof", "-top", "-sample_index=alloc_objects", "-nodecount=50", "-focus", "pipeDeadline", "-base", before, after)
	out, err := cmd.CombinedOutput()
	t.Logf("pprof pipeDeadline\n%s", out)
	if err != nil {
		t.Fatalf("pprof: %v", err)
	}
	pipe = focusedSamples(string(out)) / float64(n)
	return total, pipe
}

func focusedSamples(text string) float64 {
	const key = "accounting for "
	i := strings.Index(text, key)
	if i < 0 {
		return 0
	}
	rest := text[i+len(key):]
	field := rest
	if j := strings.IndexAny(rest, ", \n"); j >= 0 {
		field = rest[:j]
	}
	n, err := strconv.ParseFloat(strings.ReplaceAll(field, ",", ""), 64)
	if err != nil {
		return 0
	}
	return n
}
