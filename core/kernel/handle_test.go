package kernel_test

import (
	"fmt"
	"os"
	"path/filepath"
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
