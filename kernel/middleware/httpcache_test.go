package middleware_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/middleware"
)

func TestETagReturnsNotModified(t *testing.T) {
	handler := middleware.ETag(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})
	first := handler(http.NewRequest(httptest.NewRequest("GET", "/api/cached", nil)))
	etag := first.Headers().Get("ETag")
	if etag == "" || first.StatusCode() != 200 {
		t.Fatalf("first=%d etag=%q", first.StatusCode(), etag)
	}
	raw := httptest.NewRequest("GET", "/api/cached", nil)
	raw.Header.Set("If-None-Match", etag)
	second := handler(http.NewRequest(raw))
	if second.StatusCode() != 304 {
		t.Fatalf("expected 304, got %d", second.StatusCode())
	}
}

func TestCacheControlHeader(t *testing.T) {
	handler := middleware.CacheControl("public", time.Minute)(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})
	resp := handler(http.NewRequest(httptest.NewRequest("GET", "/x", nil)))
	if resp.Headers().Get("Cache-Control") == "" {
		t.Fatal("missing cache-control")
	}
}

func TestLastModified(t *testing.T) {
	mod := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	handler := middleware.LastModified(func(req *http.Request) time.Time { return mod })(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})
	first := handler(http.NewRequest(httptest.NewRequest("GET", "/x", nil)))
	if first.StatusCode() != 200 || first.Headers().Get("Last-Modified") == "" {
		t.Fatalf("first=%d lm=%q", first.StatusCode(), first.Headers().Get("Last-Modified"))
	}
	raw := httptest.NewRequest("GET", "/x", nil)
	raw.Header.Set("If-Modified-Since", mod.Format(stdhttp.TimeFormat))
	second := handler(http.NewRequest(raw))
	if second.StatusCode() != 304 {
		t.Fatalf("expected 304, got %d", second.StatusCode())
	}
}
