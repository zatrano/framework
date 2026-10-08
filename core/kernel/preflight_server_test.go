package kernel

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/framework/v3/core/contracts"
	khttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
)

// preflightAuth sits inside CORS. A browser preflight must not reach it.
type preflightAuth struct{ hit *atomic.Int32 }

func (p preflightAuth) Register(contracts.App) error { return nil }

func (p preflightAuth) Boot(app contracts.App) error {
	app.(*Application).router.Use(func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *khttp.Request) *khttp.Response {
			if req.Method() == "OPTIONS" && req.Header("Access-Control-Request-Method") != "" {
				p.hit.Add(1)
				return khttp.Abort(401, "blocked")
			}
			return next(req)
		}
	})
	return nil
}

func TestBootstrapRejectsCORSCredentialsWildcard(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("CORS_ALLOW_CREDENTIALS", "true")
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	app := NewApplication(dir)
	t.Cleanup(func() {
		if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
			_ = c.Close()
		}
	})
	err := app.Bootstrap()
	if err == nil || !strings.Contains(err.Error(), "wildcard") {
		t.Fatalf("boot err=%v", err)
	}
}

func TestRealServerPreflightAndUnmatched(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://allowed.example")

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	app := NewApplication(dir)
	t.Cleanup(func() {
		if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
			_ = c.Close()
		}
	})
	var guard atomic.Int32
	app.RegisterProviders(preflightAuth{hit: &guard})
	app.router.Get("/rota", func(*khttp.Request) *khttp.Response {
		return khttp.Text("ok")
	})
	app.router.Add("OPTIONS", "/explicit", func(*khttp.Request) *khttp.Response {
		return khttp.Text("explicit")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ln := serveTestServer(t, srv)
	addr := ln.Addr().String()

	allowed := readRaw(t, addr, "OPTIONS /missing HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nOrigin: https://allowed.example\r\nAccess-Control-Request-Method: GET\r\n\r\n")
	if allowed.StatusCode != 204 {
		t.Fatalf("allowed preflight status=%d", allowed.StatusCode)
	}
	if allowed.Header.Get("Access-Control-Allow-Origin") != "https://allowed.example" {
		t.Fatalf("acao=%q", allowed.Header.Get("Access-Control-Allow-Origin"))
	}
	if allowed.Header.Get("Access-Control-Allow-Methods") == "" ||
		allowed.Header.Get("Access-Control-Allow-Headers") == "" ||
		allowed.Header.Get("Access-Control-Max-Age") == "" {
		t.Fatalf("missing allow headers: %v", allowed.Header)
	}
	if allowed.Header.Get("Vary") != "Origin, Access-Control-Request-Method, Access-Control-Request-Headers" {
		t.Fatalf("vary=%q", allowed.Header.Get("Vary"))
	}
	if guard.Load() != 0 {
		t.Fatal("auth saw an allowed preflight")
	}

	denied := readRaw(t, addr, "OPTIONS /missing HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nOrigin: https://evil.example\r\nAccess-Control-Request-Method: GET\r\n\r\n")
	if denied.StatusCode != 404 {
		t.Fatalf("denied preflight status=%d", denied.StatusCode)
	}
	for _, h := range []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Max-Age",
	} {
		if denied.Header.Get(h) != "" {
			t.Fatalf("denied %s=%q", h, denied.Header.Get(h))
		}
	}
	if denied.Header.Get("Vary") != "Origin" {
		t.Fatalf("denied vary=%q", denied.Header.Get("Vary"))
	}
	if guard.Load() != 0 {
		t.Fatal("auth saw a denied preflight")
	}

	matchedMiss := readRaw(t, addr, "GET /missing HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nOrigin: https://allowed.example\r\n\r\n")
	if matchedMiss.StatusCode != 404 || matchedMiss.Header.Get("Vary") != "Origin" || matchedMiss.Header.Get("Access-Control-Allow-Origin") != "https://allowed.example" {
		t.Fatalf("matched 404 status=%d vary=%q acao=%q", matchedMiss.StatusCode, matchedMiss.Header.Get("Vary"), matchedMiss.Header.Get("Access-Control-Allow-Origin"))
	}
	otherMiss := readRaw(t, addr, "GET /missing HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nOrigin: https://evil.example\r\n\r\n")
	if otherMiss.StatusCode != 404 || otherMiss.Header.Get("Vary") != "Origin" || otherMiss.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("other 404 status=%d vary=%q acao=%q", otherMiss.StatusCode, otherMiss.Header.Get("Vary"), otherMiss.Header.Get("Access-Control-Allow-Origin"))
	}

	missing := readRaw(t, addr, "GET /missing HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")
	if missing.StatusCode != 404 {
		t.Fatalf("missing status=%d", missing.StatusCode)
	}
	if missing.Header.Get("Vary") != "" || missing.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("origin-less 404 vary=%q acao=%q", missing.Header.Get("Vary"), missing.Header.Get("Access-Control-Allow-Origin"))
	}
	for _, h := range []string{"X-Frame-Options", "X-Content-Type-Options", "Referrer-Policy", "Permissions-Policy", "X-Request-ID"} {
		if missing.Header.Get(h) == "" {
			t.Fatalf("404 missing %s", h)
		}
	}
	again := readRaw(t, addr, "GET /rota HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")
	if again.StatusCode != 200 {
		t.Fatalf("server after unmatched: %d", again.StatusCode)
	}

	explicit := readRaw(t, addr, "OPTIONS /explicit HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")
	if explicit.StatusCode != 200 {
		t.Fatalf("explicit OPTIONS status=%d", explicit.StatusCode)
	}
	b, err := io.ReadAll(explicit.Body)
	explicit.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "explicit" {
		t.Fatalf("explicit body=%q", b)
	}

	post := readRaw(t, addr, "POST /rota HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nContent-Length: 0\r\n\r\n")
	if post.StatusCode != 404 || post.Header.Get("Allow") != "" {
		t.Fatalf("method miss status=%d allow=%q", post.StatusCode, post.Header.Get("Allow"))
	}

	engine := readRaw(t, addr, "POST /rota HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nContent-Type: application/json\r\nContent-Length: 34603008\r\n\r\n")
	if engine.StatusCode != 413 {
		t.Fatalf("engine status=%d", engine.StatusCode)
	}
	if engine.Header.Get("X-Request-ID") != "" || engine.Header.Get("X-Frame-Options") != "" || engine.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("engine carried framework headers: %v", engine.Header)
	}
}

func readRaw(t *testing.T, addr, raw string) *http.Response {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.WriteString(c, raw); err != nil {
		t.Fatal(err)
	}
	if tc, ok := c.(*net.TCPConn); ok {
		_ = tc.CloseWrite()
	}
	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}
