package middleware_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/middleware"
)

func TestCORSWithOrigin(t *testing.T) {
	mw := middleware.CORSWith(middleware.CORSConfig{
		AllowOrigins: []string{"https://app.example"},
		AllowMethods: "GET, OPTIONS",
		AllowHeaders: "Content-Type",
		MaxAge:       60,
	})
	handler := mw(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})

	r := httptest.NewRequest(stdhttp.MethodGet, "/api/health", nil)
	r.Header.Set("Origin", "https://app.example")
	req := http.NewRequest(r)
	resp := handler(req)
	if resp.Headers().Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Fatalf("origin=%q", resp.Headers().Get("Access-Control-Allow-Origin"))
	}

	opt := httptest.NewRequest(stdhttp.MethodOptions, "/api/health", nil)
	opt.Header.Set("Origin", "https://app.example")
	preflight := handler(http.NewRequest(opt))
	if preflight.StatusCode() != 204 {
		t.Fatalf("status=%d", preflight.StatusCode())
	}
}

func TestCORSWildcard(t *testing.T) {
	handler := middleware.CORS(func(req *http.Request) *http.Response {
		return http.NoContent()
	})
	r := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	resp := handler(http.NewRequest(r))
	if resp.Headers().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("expected *")
	}
}

func TestCORSCredentialsNotWithWildcard(t *testing.T) {
	mw := middleware.CORSWith(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	})
	handler := mw(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})
	r := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://evil.example")
	resp := handler(http.NewRequest(r))
	if resp.Headers().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("origin=%q", resp.Headers().Get("Access-Control-Allow-Origin"))
	}
	if resp.Headers().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("credentials must not be set with wildcard origin")
	}
}
