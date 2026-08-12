package trustedproxy_test

import (
	stdhttp "net/http"
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/trustedproxy"
)

func TestResolveIgnoresForwardedWhenUntrusted(t *testing.T) {
	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "203.0.113.10:1234"
	raw.Header.Set("X-Forwarded-For", "198.51.100.1")
	req := http.NewRequest(raw)

	ip := trustedproxy.Resolve(req, false, nil)
	if ip != "203.0.113.10" {
		t.Fatalf("got %q", ip)
	}
}

func TestResolveTrustsForwardedWhenTrusted(t *testing.T) {
	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "127.0.0.1:1234"
	raw.Header.Set("X-Forwarded-For", "198.51.100.1")
	req := http.NewRequest(raw)

	ip := trustedproxy.Resolve(req, true, nil)
	if ip != "198.51.100.1" {
		t.Fatalf("got %q", ip)
	}
}

func TestMiddlewareSetsForwardedProtoHost(t *testing.T) {
	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "127.0.0.1:1234"
	raw.Host = "localhost"
	raw.Header.Set("X-Forwarded-Proto", "https")
	raw.Header.Set("X-Forwarded-Host", "app.example.test")
	req := http.NewRequest(raw)

	mw := trustedproxy.Middleware("*")
	handler := mw(func(r *http.Request) *http.Response {
		if !r.Secure() || r.Host() != "app.example.test" || r.Scheme() != "https" {
			t.Fatalf("scheme=%s host=%s secure=%v", r.Scheme(), r.Host(), r.Secure())
		}
		return http.JSON(map[string]any{"ok": true})
	})
	resp := handler(req)
	if resp.StatusCode() != 200 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
}
