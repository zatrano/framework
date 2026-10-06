package middleware_test

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/middleware"
)

func TestSecurityHeadersDoNotBuildHeaderMap(t *testing.T) {
	handler := middleware.SecurityHeaders(middleware.RequestID(func(req *http.Request) *http.Response {
		return http.Text("ok")
	}))
	resp := handler(http.RequestFromHTTP(httptest.NewRequest("GET", "/", nil)))
	if resp.HeaderMapBuilt() {
		t.Fatal("security and request id built a header map")
	}
	if resp.GetHeader("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff")
	}
	if resp.GetHeader("X-Request-ID") == "" {
		t.Fatal("missing request id")
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := middleware.SecurityHeaders(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})
	resp := handler(http.RequestFromHTTP(httptest.NewRequest("GET", "/api/health", nil)))
	if resp.Headers().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff")
	}
	if resp.Headers().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Fatal("missing frame options")
	}
	if resp.Headers().Get("Strict-Transport-Security") != "" {
		t.Fatal("default SecurityHeaders must not emit HSTS")
	}
}

func TestSecurityHeadersHSTSOnlyOnHTTPS(t *testing.T) {
	h := middleware.SecurityHeadersWith(middleware.SecurityHeaderConfig{EnableHSTSOnHTTPS: true})
	httpReq := http.RequestFromHTTP(httptest.NewRequest("GET", "http://app.example/", nil))
	httpResp := h(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})(httpReq)
	if httpResp.Headers().Get("Strict-Transport-Security") != "" {
		t.Fatal("HSTS on HTTP")
	}

	raw := httptest.NewRequest("GET", "https://app.example/", nil)
	raw.TLS = &tls.ConnectionState{}
	httpsResp := h(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})(http.RequestFromHTTP(raw))
	if httpsResp.Headers().Get("Strict-Transport-Security") != "max-age=31536000" {
		t.Fatalf("HSTS=%q", httpsResp.Headers().Get("Strict-Transport-Security"))
	}
}

func TestSecurityHeadersHSTSUsesTrustedForwardedProto(t *testing.T) {
	h := middleware.SecurityHeadersWith(middleware.SecurityHeaderConfig{EnableHSTSOnHTTPS: true})
	req := http.RequestFromHTTP(httptest.NewRequest("GET", "http://app.example/", nil))
	req.Set("_forwarded_proto", "https")
	resp := h(func(r *http.Request) *http.Response {
		return http.Text("ok")
	})(req)
	if resp.Headers().Get("Strict-Transport-Security") != "max-age=31536000" {
		t.Fatal("trusted forwarded proto must enable HSTS")
	}
}
