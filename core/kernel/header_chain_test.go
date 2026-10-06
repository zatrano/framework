package kernel

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	khttp "github.com/zatrano/framework/v3/core/kernel/http"
)

func TestDefaultChainDoesNotBuildHeaderMap(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://allowed.example")
	t.Setenv("TRUSTED_PROXIES", "")
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
	app.router.Get("/rota", func(req *khttp.Request) *khttp.Response {
		if req.HeaderCacheBuilt() {
			t.Error("default chain built the request header map")
		}
		return khttp.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ln := serveTestServer(t, srv)
	resp := readRaw(t, ln.Addr().String(), "GET /rota HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || string(body) != "ok" {
		t.Fatalf("status=%d body=%q", resp.StatusCode, body)
	}
}
