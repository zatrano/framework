package kernel_test

import (
	"crypto/tls"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/kernel/http"
)

func bootProductionApp(t *testing.T) *kernel.Application {
	t.Helper()
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("SESSION_SECURE", "")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	return app
}

func cookieNamed(t *testing.T, rec *httptest.ResponseRecorder, name string) *stdhttp.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("missing cookie %s: %v", name, rec.Result().Cookies())
	return nil
}

func TestProductionHTTPSSetsHSTS(t *testing.T) {
	app := bootProductionApp(t)
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	raw := httptest.NewRequest(stdhttp.MethodGet, "https://app.example/ok", nil)
	raw.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, raw)
	hsts := rec.Header().Get("Strict-Transport-Security")
	if hsts != "max-age=31536000" {
		t.Fatalf("HSTS=%q", hsts)
	}
	if strings.Contains(strings.ToLower(hsts), "includesubdomains") || strings.Contains(strings.ToLower(hsts), "preload") {
		t.Fatalf("default HSTS must not include includeSubDomains/preload: %q", hsts)
	}
}

func TestProductionHTTPDoesNotSetHSTS(t *testing.T) {
	app := bootProductionApp(t)
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "http://app.example/ok", nil))
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("HSTS must not be sent on plain HTTP")
	}
}

func TestProductionUntrustedForwardedProtoDoesNotSetHSTS(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")
	app := bootProductionApp(t)
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	raw := httptest.NewRequest(stdhttp.MethodGet, "http://app.example/ok", nil)
	raw.RemoteAddr = "203.0.113.10:1234"
	raw.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, raw)
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("untrusted X-Forwarded-Proto must not enable HSTS")
	}
}

func TestProductionTrustedProxyHTTPSSetsHSTS(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "192.0.2.1")
	app := bootProductionApp(t)
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	raw := httptest.NewRequest(stdhttp.MethodGet, "http://app.example/ok", nil)
	raw.RemoteAddr = "192.0.2.1:443"
	raw.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, raw)
	if rec.Header().Get("Strict-Transport-Security") != "max-age=31536000" {
		t.Fatalf("HSTS=%q", rec.Header().Get("Strict-Transport-Security"))
	}
}

func TestDevelopmentTLSDoesNotSetHSTS(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	raw := httptest.NewRequest(stdhttp.MethodGet, "https://app.example/ok", nil)
	raw.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, raw)
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("development must not emit default HSTS")
	}
}

func TestProductionJarCookieIsSecure(t *testing.T) {
	app := bootProductionApp(t)
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		req.Cookies().Queue("sid", "abc", 10)
		return http.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "http://app.example/ok", nil))
	c := cookieNamed(t, rec, "sid")
	if !c.Secure {
		t.Fatal("production framework jar cookie must be Secure")
	}
	if !c.HttpOnly {
		t.Fatal("HttpOnly must be preserved")
	}
	if c.SameSite != stdhttp.SameSiteLaxMode {
		t.Fatalf("SameSite=%v", c.SameSite)
	}
}

func TestProductionAppCookieIsNotForcedSecure(t *testing.T) {
	app := bootProductionApp(t)
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok").Cookie("pref", "dark")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "http://app.example/ok", nil))
	c := cookieNamed(t, rec, "pref")
	if c.Secure {
		t.Fatal("application Cookie() must not be silently forced Secure")
	}
}

func TestProductionExplicitCookieOptionsNotOverridden(t *testing.T) {
	app := bootProductionApp(t)
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok").WithCookieOptions("pref", "dark", http.CookieOptions{
			Path:     "/",
			Secure:   false,
			HTTPOnly: false,
			SameSite: stdhttp.SameSiteNoneMode,
		})
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "http://app.example/ok", nil))
	c := cookieNamed(t, rec, "pref")
	if c.Secure || c.HttpOnly || c.SameSite != stdhttp.SameSiteNoneMode {
		t.Fatalf("explicit options overridden: %+v", c)
	}
}
