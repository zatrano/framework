package middleware_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/middleware"
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
	req := http.RequestFromHTTP(r)
	resp := handler(req)
	if resp.Headers().Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Fatalf("origin=%q", resp.Headers().Get("Access-Control-Allow-Origin"))
	}

	opt := httptest.NewRequest(stdhttp.MethodOptions, "/api/health", nil)
	opt.Header.Set("Origin", "https://app.example")
	opt.Header.Set("Access-Control-Request-Method", "GET")
	preflight := handler(http.RequestFromHTTP(opt))
	if preflight.StatusCode() != 204 {
		t.Fatalf("status=%d", preflight.StatusCode())
	}
	if preflight.Headers().Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Fatalf("acao=%q", preflight.Headers().Get("Access-Control-Allow-Origin"))
	}
	if preflight.Headers().Get("Access-Control-Allow-Methods") != "GET, OPTIONS" {
		t.Fatalf("methods=%q", preflight.Headers().Get("Access-Control-Allow-Methods"))
	}
	if preflight.Headers().Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Fatalf("headers=%q", preflight.Headers().Get("Access-Control-Allow-Headers"))
	}
	if preflight.Headers().Get("Access-Control-Max-Age") != "60" {
		t.Fatalf("max-age=%q", preflight.Headers().Get("Access-Control-Max-Age"))
	}
	if preflight.Headers().Get("Vary") != "Origin, Access-Control-Request-Method, Access-Control-Request-Headers" {
		t.Fatalf("vary=%q", preflight.Headers().Get("Vary"))
	}

	plain := httptest.NewRequest(stdhttp.MethodOptions, "/api/health", nil)
	plain.Header.Set("Origin", "https://app.example")
	hit := handler(http.RequestFromHTTP(plain))
	if hit.StatusCode() != 200 {
		t.Fatalf("non-preflight OPTIONS status=%d", hit.StatusCode())
	}

	deniedHit := false
	denied := mw(func(req *http.Request) *http.Response {
		deniedHit = true
		return http.Text("no")
	})
	bad := httptest.NewRequest(stdhttp.MethodOptions, "/api/health", nil)
	bad.Header.Set("Origin", "https://evil.example")
	bad.Header.Set("Access-Control-Request-Method", "POST")
	deniedResp := denied(http.RequestFromHTTP(bad))
	if deniedHit {
		t.Fatal("disallowed preflight must not reach later middleware")
	}
	if deniedResp.StatusCode() != 404 {
		t.Fatalf("denied status=%d", deniedResp.StatusCode())
	}
	for _, h := range []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Max-Age",
	} {
		if deniedResp.Headers().Get(h) != "" {
			t.Fatalf("denied %s=%q", h, deniedResp.Headers().Get(h))
		}
	}
}

func TestCORSWildcard(t *testing.T) {
	handler := middleware.CORS(func(req *http.Request) *http.Response {
		return http.NoContent()
	})
	r := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	resp := handler(http.RequestFromHTTP(r))
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
	resp := handler(http.RequestFromHTTP(r))
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
	resp := handler(http.RequestFromHTTP(r))
	if resp.Headers().Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Fatalf("origin=%q", resp.Headers().Get("Access-Control-Allow-Origin"))
	}
	if resp.Headers().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("expected credentials")
	}

	bad := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	bad.Header.Set("Origin", "https://evil.example")
	resp2 := handler(http.RequestFromHTTP(bad))
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
	resp := handler(http.RequestFromHTTP(r))
	if resp.Headers().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("production must not allow wildcard CORS")
	}

	mw = middleware.CORSWith(middleware.CORSConfig{AllowOrigins: []string{"*"}, Production: true})
	handler = mw(func(req *http.Request) *http.Response {
		return http.NoContent()
	})
	resp = handler(http.RequestFromHTTP(r))
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
	resp := handler(http.RequestFromHTTP(r))
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
	resp := handler(http.RequestFromHTTP(r))
	if resp.Headers().Get("Access-Control-Allow-Origin") == "*" {
		t.Fatal("CORS production snapshot must ignore later APP_ENV")
	}
}

func TestCORSOriginTable(t *testing.T) {
	mw := middleware.CORSWith(middleware.CORSConfig{
		AllowOrigins: []string{"https://app.example"},
		AllowMethods: "GET, POST, OPTIONS",
		AllowHeaders: "Content-Type, Authorization",
		MaxAge:       600,
	})
	nextHits := 0
	handler := mw(func(req *http.Request) *http.Response {
		nextHits++
		if req.Path() == "/missing" {
			return http.Abort(404, "Not Found")
		}
		return http.Text("ok")
	})

	cases := []struct {
		name       string
		method     string
		path       string
		origins    []string
		acrMethod  string
		acrHeaders string
		status     int
		acao       string
		allowHdr   string
		vary       string
		hit        bool
	}{
		{name: "exact", method: "GET", path: "/", origins: []string{"https://app.example"}, status: 200, acao: "https://app.example", allowHdr: "Content-Type, Authorization", vary: "Origin", hit: true},
		{name: "null", method: "GET", path: "/", origins: []string{"null"}, status: 200, vary: "Origin", hit: true},
		{name: "scheme case", method: "GET", path: "/", origins: []string{"HTTPS://app.example"}, status: 200, vary: "Origin", hit: true},
		{name: "host case", method: "GET", path: "/", origins: []string{"https://APP.example"}, status: 200, vary: "Origin", hit: true},
		{name: "port", method: "GET", path: "/", origins: []string{"https://app.example:8443"}, status: 200, vary: "Origin", hit: true},
		{name: "slash", method: "GET", path: "/", origins: []string{"https://app.example/"}, status: 200, vary: "Origin", hit: true},
		{name: "multi", method: "GET", path: "/", origins: []string{"https://app.example", "https://other.example"}, status: 200, vary: "Origin", hit: true},
		{name: "options no origin", method: "OPTIONS", path: "/", status: 200, hit: true},
		{name: "404 match", method: "GET", path: "/missing", origins: []string{"https://app.example"}, status: 404, acao: "https://app.example", allowHdr: "Content-Type, Authorization", vary: "Origin", hit: true},
		{name: "404 other", method: "GET", path: "/missing", origins: []string{"https://evil.example"}, status: 404, vary: "Origin", hit: true},
		{name: "preflight intersect", method: "OPTIONS", path: "/", origins: []string{"https://app.example"}, acrMethod: "POST", acrHeaders: "authorization, X-Evil", status: 204, acao: "https://app.example", allowHdr: "Authorization", vary: "Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
		{name: "preflight empty intersect", method: "OPTIONS", path: "/", origins: []string{"https://app.example"}, acrMethod: "POST", acrHeaders: "X-Evil", status: 204, acao: "https://app.example", vary: "Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
		{name: "preflight no acr headers", method: "OPTIONS", path: "/", origins: []string{"https://app.example"}, acrMethod: "GET", status: 204, acao: "https://app.example", allowHdr: "Content-Type, Authorization", vary: "Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
		{name: "preflight denied", method: "OPTIONS", path: "/", origins: []string{"https://evil.example"}, acrMethod: "GET", status: 404, vary: "Origin"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := nextHits
			raw := httptest.NewRequest(tc.method, tc.path, nil)
			for _, o := range tc.origins {
				raw.Header.Add("Origin", o)
			}
			if tc.acrMethod != "" {
				raw.Header.Set("Access-Control-Request-Method", tc.acrMethod)
			}
			if tc.acrHeaders != "" {
				raw.Header.Set("Access-Control-Request-Headers", tc.acrHeaders)
			}
			resp := handler(http.RequestFromHTTP(raw))
			if resp.StatusCode() != tc.status {
				t.Fatalf("status=%d", resp.StatusCode())
			}
			if resp.Headers().Get("Access-Control-Allow-Origin") != tc.acao {
				t.Fatalf("acao=%q", resp.Headers().Get("Access-Control-Allow-Origin"))
			}
			if resp.Headers().Get("Access-Control-Allow-Headers") != tc.allowHdr {
				t.Fatalf("allow-headers=%q", resp.Headers().Get("Access-Control-Allow-Headers"))
			}
			if resp.Headers().Get("Vary") != tc.vary {
				t.Fatalf("vary=%q", resp.Headers().Get("Vary"))
			}
			hit := nextHits > before
			if hit != tc.hit {
				t.Fatalf("next hit=%v", hit)
			}
			if tc.status == 204 && resp.Headers().Get("Access-Control-Max-Age") != "600" {
				t.Fatalf("max-age=%q", resp.Headers().Get("Access-Control-Max-Age"))
			}
		})
	}
}

func TestCORSMaxAgeFromEnv(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example")
	t.Setenv("CORS_MAX_AGE", "")
	mw := middleware.CORSFromEnv("production")
	handler := mw(func(req *http.Request) *http.Response { return http.NoContent() })
	raw := httptest.NewRequest(stdhttp.MethodOptions, "/", nil)
	raw.Header.Set("Origin", "https://app.example")
	raw.Header.Set("Access-Control-Request-Method", "GET")
	resp := handler(http.RequestFromHTTP(raw))
	if resp.Headers().Get("Access-Control-Max-Age") != "600" {
		t.Fatalf("default max-age=%q", resp.Headers().Get("Access-Control-Max-Age"))
	}

	t.Setenv("CORS_MAX_AGE", "120")
	mw = middleware.CORSFromEnv("production")
	handler = mw(func(req *http.Request) *http.Response { return http.NoContent() })
	resp = handler(http.RequestFromHTTP(raw))
	if resp.Headers().Get("Access-Control-Max-Age") != "120" {
		t.Fatalf("max-age=%q", resp.Headers().Get("Access-Control-Max-Age"))
	}
}

func TestCORSCredentialsWildcardIsBootError(t *testing.T) {
	t.Setenv("CORS_ALLOW_CREDENTIALS", "true")
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")
	if err := middleware.ValidateCORSEnv("local"); err == nil {
		t.Fatal("explicit wildcard")
	}
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example,*")
	if err := middleware.ValidateCORSEnv("production"); err == nil {
		t.Fatal("mixed wildcard")
	}
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	if err := middleware.ValidateCORSEnv("local"); err == nil {
		t.Fatal("implicit development wildcard")
	}
	if err := middleware.ValidateCORSEnv("production"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example")
	if err := middleware.ValidateCORSEnv("production"); err != nil {
		t.Fatal(err)
	}
}

func TestCORSUnconfiguredAddsNothing(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("CORS_ALLOW_CREDENTIALS", "false")
	mw := middleware.CORSFromEnv("production")
	handler := mw(func(req *http.Request) *http.Response { return http.Text("ok") })
	raw := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.Header.Set("Origin", "https://app.example")
	resp := handler(http.RequestFromHTTP(raw))
	if resp.Headers().Get("Vary") != "" || resp.Headers().Get("Access-Control-Allow-Origin") != "" || resp.Headers().Get("Access-Control-Allow-Methods") != "" {
		t.Fatalf("headers=%v", resp.Headers())
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
	resp := handler(http.RequestFromHTTP(r))
	if resp.Headers().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("null Origin must not match")
	}
}

func TestStackLoggerForceJSONDomainMethodOverride(t *testing.T) {
	inner := func(req *http.Request) *http.Response {
		return http.Text(req.Header("Accept") + ":" + req.Method())
	}
	h := middleware.Stack(inner, middleware.ForceJSON, middleware.Logger, middleware.MethodOverride)
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", nil)
	raw.Header.Set("X-HTTP-Method-Override", "PUT")
	resp := h(http.RequestFromHTTP(raw))
	if !strings.Contains(string(resp.Content()), "application/json") || !strings.Contains(string(resp.Content()), "PUT") {
		t.Fatalf("body=%s", resp.Content())
	}
	ok := middleware.Domain("example.com", "*.app.test")(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	good := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	good.Host = "example.com"
	if ok(http.RequestFromHTTP(good)).StatusCode() != 200 {
		t.Fatal("exact host")
	}
	sub := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	sub.Host = "api.app.test:443"
	if ok(http.RequestFromHTTP(sub)).StatusCode() != 200 {
		t.Fatal("wildcard host")
	}
	bad := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	bad.Host = "evil.test"
	if ok(http.RequestFromHTTP(bad)).StatusCode() != 404 {
		t.Fatal("rejected host")
	}
	passthrough := middleware.Domain()(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	if passthrough(http.RequestFromHTTP(good)).StatusCode() != 200 {
		t.Fatal("empty domain list")
	}
}
