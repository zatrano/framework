package kernel_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/kernel"
)

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

func TestTH01PublicFileSecurityHeadersAndRequestID(t *testing.T) {
	app := bootPublicApp(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/app.css", nil)
	req.Header.Set("X-Request-ID", "static-req-1")
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Body.String() != "body{}" {
		t.Fatalf("body=%q", rec.Body.String())
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff: %v", rec.Header())
	}
	if rec.Header().Get("X-Frame-Options") == "" {
		t.Fatalf("missing X-Frame-Options: %v", rec.Header())
	}
	if rec.Header().Get("X-Request-ID") != "static-req-1" {
		t.Fatalf("request id=%q", rec.Header().Get("X-Request-ID"))
	}
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("development public files must not emit default HSTS")
	}
	if cd := rec.Header().Get("Content-Disposition"); strings.Contains(strings.ToLower(cd), "attachment") {
		t.Fatalf("public asset must not be an attachment: %q", cd)
	}
}

func TestTH03PostPublicFileDoesNotServeFile(t *testing.T) {
	app := bootPublicApp(t)

	post := httptest.NewRecorder()
	app.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/app.css", nil))
	if post.Code == http.StatusOK && post.Body.String() == "body{}" {
		t.Fatal("POST served public file, bypassing method policy")
	}

	override := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/app.css", nil)
	req.Header.Set("X-HTTP-Method-Override", "DELETE")
	app.ServeHTTP(override, req)
	if override.Code == http.StatusOK && override.Body.String() == "body{}" {
		t.Fatal("method-overridden POST served public file")
	}
}

func TestPublicFileHeadKeepsSecurityHeaders(t *testing.T) {
	app := bootPublicApp(t)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/app.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff on HEAD: %v", rec.Header())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD body=%q", rec.Body.String())
	}
}
