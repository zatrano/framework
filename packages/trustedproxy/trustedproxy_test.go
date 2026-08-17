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

func TestFromEnvIgnoresStarInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("TRUSTED_PROXIES", "*")
	t.Setenv("TRUST_PROXIES_ALLOW_STAR", "")

	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "203.0.113.10:1234"
	raw.Host = "localhost"
	raw.Header.Set("X-Forwarded-For", "198.51.100.1")
	raw.Header.Set("X-Forwarded-Proto", "https")
	raw.Header.Set("X-Forwarded-Host", "evil.example")
	req := http.NewRequest(raw)

	mw := trustedproxy.FromEnv()
	handler := mw(func(r *http.Request) *http.Response {
		if r.Secure() {
			t.Fatal("production must ignore TRUSTED_PROXIES=* without allow flag")
		}
		if r.Host() == "evil.example" {
			t.Fatal("forwarded host must not apply")
		}
		return http.JSON(map[string]any{"ok": true})
	})
	_ = handler(req)
}

func TestFromEnvAllowsStarWhenExplicit(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("TRUSTED_PROXIES", "*")
	t.Setenv("TRUST_PROXIES_ALLOW_STAR", "true")

	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "203.0.113.10:1234"
	raw.Host = "localhost"
	raw.Header.Set("X-Forwarded-Proto", "https")
	raw.Header.Set("X-Forwarded-Host", "app.example.test")
	req := http.NewRequest(raw)

	mw := trustedproxy.FromEnv()
	handler := mw(func(r *http.Request) *http.Response {
		if !r.Secure() || r.Host() != "app.example.test" {
			t.Fatalf("scheme secure=%v host=%s", r.Secure(), r.Host())
		}
		return http.JSON(map[string]any{"ok": true})
	})
	_ = handler(req)
}
