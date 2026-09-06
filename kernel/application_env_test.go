package kernel_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
	"github.com/zatrano/framework/v2/kernel/env"
	"github.com/zatrano/framework/v2/kernel/http"
)

func corsOrigin(rec *httptest.ResponseRecorder) string {
	return rec.Header().Get("Access-Control-Allow-Origin")
}

func TestProductionCasingSharesEnvironmentTruth(t *testing.T) {
	for _, raw := range []string{"production", "Production", " PRODUCTION "} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("APP_ENV", raw)
			t.Setenv("APP_DEBUG", "false")
			t.Setenv("APP_KEY", strings.Repeat("s", 32))
			t.Setenv("CORS_ALLOWED_ORIGINS", "")
			app := kernel.NewApplication(t.TempDir())
			t.Cleanup(func() { closeAppLog(t, app) })
			app.Router().Get("/ok", func(req *http.Request) *http.Response {
				req.Cookies().Queue("sid", "1", 10)
				return http.Text("ok")
			})
			if err := app.Bootstrap(); err != nil {
				t.Fatal(err)
			}
			if !app.IsProduction() {
				t.Fatalf("IsProduction=false for APP_ENV=%q env=%q", raw, app.Environment())
			}
			if app.Environment() != "production" {
				t.Fatalf("environment=%q", app.Environment())
			}

			if err := env.Set("APP_ENV", "local"); err != nil {
				t.Fatal(err)
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(stdhttp.MethodGet, "/ok", nil)
			req.Header.Set("Origin", "https://evil.example")
			app.ServeHTTP(rec, req)
			if corsOrigin(rec) == "*" {
				t.Fatal("production CORS must not emit wildcard after APP_ENV mutation")
			}
			c := cookieNamed(t, rec, "sid")
			if !c.Secure {
				t.Fatal("production cookie policy must not follow mutated APP_ENV")
			}
		})
	}
}

func TestStagingCORSHasNoImplicitWildcard(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.IsProduction() {
		t.Fatal("staging must not be treated as production")
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "/ok", nil)
	req.Header.Set("Origin", "https://app.example")
	app.ServeHTTP(rec, req)
	if corsOrigin(rec) == "*" {
		t.Fatal("staging must not default CORS to *")
	}
}

func TestDevelopmentCORSKeepsImplicitWildcard(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if app.IsProduction() {
		t.Fatal("development must not be production")
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "/ok", nil)
	req.Header.Set("Origin", "https://app.example")
	app.ServeHTTP(rec, req)
	if corsOrigin(rec) != "*" {
		t.Fatalf("development CORS origin=%q", corsOrigin(rec))
	}
}

func TestProductionRejectsExplicitCORSWildcard(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("APP_KEY", strings.Repeat("s", 32))
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ok", func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "/ok", nil)
	req.Header.Set("Origin", "https://evil.example")
	app.ServeHTTP(rec, req)
	if corsOrigin(rec) == "*" || corsOrigin(rec) == "https://evil.example" {
		t.Fatalf("production CORS_ALLOWED_ORIGINS=* must be sanitized, got %q", corsOrigin(rec))
	}
}
