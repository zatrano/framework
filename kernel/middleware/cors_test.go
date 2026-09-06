package middleware_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/middleware"
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
	if resp.Headers().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("origin=%q — wildcard must be dropped when credentials enabled", resp.Headers().Get("Access-Control-Allow-Origin"))
	}
	if resp.Headers().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("credentials must not be set with wildcard origin")
	}
}

func TestCORSWildcardCredentials(t *testing.T) {
	mw := middleware.CORSWith(middleware.CORSConfig{
		AllowOrigins:     []string{"https://app.example"},
		AllowCredentials: true,
	})
	handler := mw(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})
	r := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://app.example")
	resp := handler(http.NewRequest(r))
	if resp.Headers().Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Fatalf("origin=%q", resp.Headers().Get("Access-Control-Allow-Origin"))
	}
	if resp.Headers().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("expected credentials")
	}

	bad := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	bad.Header.Set("Origin", "https://evil.example")
	resp2 := handler(http.NewRequest(bad))
	if resp2.Headers().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("unknown origin must not be reflected")
	}
}

func TestCORSProductionNoWildcardDefault(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	mw := middleware.CORSFromEnv("production")
	handler := mw(func(req *http.Request) *http.Response {
		return http.NoContent()
	})
	r := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://evil.example")
	resp := handler(http.NewRequest(r))
	if resp.Headers().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("production must not allow wildcard CORS")
	}

	mw = middleware.CORSWith(middleware.CORSConfig{AllowOrigins: []string{"*"}, Production: true})
	handler = mw(func(req *http.Request) *http.Response {
		return http.NoContent()
	})
	resp = handler(http.NewRequest(r))
	if resp.Headers().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("production must sanitize explicit wildcard")
	}
}

func TestCORSFromEnvStagingNoImplicitWildcard(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	mw := middleware.CORSFromEnv("staging")
	handler := mw(func(req *http.Request) *http.Response {
		return http.NoContent()
	})
	r := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://app.example")
	resp := handler(http.NewRequest(r))
	if resp.Headers().Get("Access-Control-Allow-Origin") == "*" {
		t.Fatal("staging must not default CORS to *")
	}
}

func TestCORSFromEnvUsesSnapshotNotProcessEnv(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	mw := middleware.CORSFromEnv("production")
	t.Setenv("APP_ENV", "development")
	handler := mw(func(req *http.Request) *http.Response {
		return http.NoContent()
	})
	r := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://evil.example")
	resp := handler(http.NewRequest(r))
	if resp.Headers().Get("Access-Control-Allow-Origin") == "*" {
		t.Fatal("CORS production snapshot must ignore later APP_ENV")
	}
}

func TestCORSNullOrigin(t *testing.T) {
	mw := middleware.CORSWith(middleware.CORSConfig{
		AllowOrigins: []string{"https://app.example"},
	})
	handler := mw(func(req *http.Request) *http.Response {
		return http.NoContent()
	})
	r := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	r.Header.Set("Origin", "null")
	resp := handler(http.NewRequest(r))
	if resp.Headers().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("null Origin must not match")
	}
}
