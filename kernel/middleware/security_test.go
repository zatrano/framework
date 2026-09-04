package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/middleware"
)

func TestSecurityHeaders(t *testing.T) {
	handler := middleware.SecurityHeaders(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})
	resp := handler(http.NewRequest(httptest.NewRequest("GET", "/api/health", nil)))
	if resp.Headers().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff")
	}
	if resp.Headers().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Fatal("missing frame options")
	}
}
