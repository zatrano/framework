package trustedproxy_test

import (
	"net"
	stdhttp "net/http"
	"testing"

	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/middleware"
	"github.com/zatrano/framework/kernel/trustedproxy"
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

func TestResolveUsesRightmostClientWhenProxyAppends(t *testing.T) {
	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "192.0.2.1:443"
	// Attacker prepends 10.0.0.1; trusted proxy appends the real client.
	raw.Header.Set("X-Forwarded-For", "10.0.0.1, 198.51.100.7")
	req := http.NewRequest(raw)

	_, n, err := net.ParseCIDR("192.0.2.1/32")
	if err != nil {
		t.Fatal(err)
	}
	ip := trustedproxy.Resolve(req, false, []*net.IPNet{n})
	if ip != "198.51.100.7" {
		t.Fatalf("spoofed leftmost XFF was trusted: got %q", ip)
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

func TestParseStarAcceptedPolicyRejectsProduction(t *testing.T) {
	cfg, err := trustedproxy.Parse("*")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.TrustAll {
		t.Fatal("parser must keep * as TrustAll, not drop it")
	}
	if err := cfg.Validate(false); err != nil {
		t.Fatalf("development policy rejected *: %v", err)
	}
	if err := cfg.Validate(true); err == nil {
		t.Fatal("production policy must reject wildcard")
	}
}

func TestParseMalformedFails(t *testing.T) {
	if _, err := trustedproxy.Parse("not-a-proxy"); err == nil {
		t.Fatal("malformed entry was accepted")
	}
}

func TestFromEnvProductionStarFailsEvenWithAllowFlag(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "*")
	t.Setenv("TRUST_PROXIES_ALLOW_STAR", "true")
	if _, err := trustedproxy.FromEnv(true); err == nil {
		t.Fatal("production FromEnv accepted TRUSTED_PROXIES=*")
	}
}

func TestFromEnvDevelopmentStarTrustsForwarded(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "*")
	mw, err := trustedproxy.FromEnv(false)
	if err != nil {
		t.Fatal(err)
	}

	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "203.0.113.10:1234"
	raw.Host = "localhost"
	raw.Header.Set("X-Forwarded-Proto", "https")
	raw.Header.Set("X-Forwarded-Host", "app.example.test")
	req := http.NewRequest(raw)

	handler := mw(func(r *http.Request) *http.Response {
		if !r.Secure() || r.Host() != "app.example.test" {
			t.Fatalf("scheme secure=%v host=%s", r.Secure(), r.Host())
		}
		return http.JSON(map[string]any{"ok": true})
	})
	_ = handler(req)
}

func TestFromEnvProductionSpecificProxyTrustsForwarded(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "192.0.2.1")
	mw, err := trustedproxy.FromEnv(true)
	if err != nil {
		t.Fatal(err)
	}

	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "192.0.2.1:443"
	raw.Header.Set("X-Forwarded-For", "198.51.100.7")
	req := http.NewRequest(raw)
	handler := mw(func(r *http.Request) *http.Response {
		if r.IP() != "198.51.100.7" {
			t.Fatalf("got %q", r.IP())
		}
		return http.JSON(map[string]any{"ok": true})
	})
	_ = handler(req)
}

func TestFromEnvProductionEmptyDoesNotTrustXFF(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")
	mw, err := trustedproxy.FromEnv(true)
	if err != nil {
		t.Fatal(err)
	}

	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "203.0.113.10:1234"
	raw.Header.Set("X-Forwarded-For", "198.51.100.1")
	req := http.NewRequest(raw)
	handler := mw(func(r *http.Request) *http.Response {
		if r.IP() != "203.0.113.10" {
			t.Fatalf("empty TRUSTED_PROXIES trusted XFF: got %q", r.IP())
		}
		return http.JSON(map[string]any{"ok": true})
	})
	_ = handler(req)
}

func TestFromEnvProductionMixedStarFails(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "192.0.2.1, *")
	if _, err := trustedproxy.FromEnv(true); err == nil {
		t.Fatal("mixed * in production was accepted")
	}
}

func cidr(t *testing.T, s string) *net.IPNet {
	t.Helper()
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestResolveSingleTrustedProxy(t *testing.T) {
	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "192.0.2.1:443"
	raw.Header.Set("X-Forwarded-For", "198.51.100.7")
	ip := trustedproxy.Resolve(http.NewRequest(raw), false, []*net.IPNet{cidr(t, "192.0.2.1/32")})
	if ip != "198.51.100.7" {
		t.Fatalf("got %q", ip)
	}
}

func TestResolveMultipleTrustedHops(t *testing.T) {
	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "192.0.2.2:443"
	raw.Header.Set("X-Forwarded-For", "10.0.0.1, 198.51.100.7, 192.0.2.1")
	ip := trustedproxy.Resolve(http.NewRequest(raw), false, []*net.IPNet{
		cidr(t, "192.0.2.1/32"),
		cidr(t, "192.0.2.2/32"),
	})
	if ip != "198.51.100.7" {
		t.Fatalf("got %q", ip)
	}
}

func TestResolveHopOverflowPrepend(t *testing.T) {
	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "192.0.2.1:443"
	raw.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3, 10.0.0.4, 198.51.100.7")
	ip := trustedproxy.Resolve(http.NewRequest(raw), false, []*net.IPNet{cidr(t, "192.0.2.1/32")})
	if ip != "198.51.100.7" {
		t.Fatalf("got %q", ip)
	}
}

func TestResolveMalformedXFF(t *testing.T) {
	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "192.0.2.1:443"
	raw.Header.Set("X-Forwarded-For", "not-an-ip, unknown, 198.51.100.7")
	ip := trustedproxy.Resolve(http.NewRequest(raw), false, []*net.IPNet{cidr(t, "192.0.2.1/32")})
	if ip != "198.51.100.7" {
		t.Fatalf("got %q", ip)
	}

	raw.Header.Set("X-Forwarded-For", "garbage")
	ip = trustedproxy.Resolve(http.NewRequest(raw), false, []*net.IPNet{cidr(t, "192.0.2.1/32")})
	if ip != "192.0.2.1" {
		t.Fatalf("malformed-only XFF got %q", ip)
	}
}

func TestAllowIPNotBypassedByPrependedXFF(t *testing.T) {
	h := trustedproxy.Middleware("192.0.2.1")(middleware.AllowIP("10.0.0.1")(func(req *http.Request) *http.Response {
		return http.Text("ok")
	}))
	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.RemoteAddr = "192.0.2.1:443"
	raw.Header.Set("X-Forwarded-For", "10.0.0.1, 203.0.113.50")
	resp := h(http.NewRequest(raw))
	if resp.StatusCode() != 403 {
		t.Fatalf("AllowIP bypassed via XFF prepend: status=%d", resp.StatusCode())
	}
}
