package csrf_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel/cookie"
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/middleware"
	"github.com/zatrano/framework/v2/kernel/middleware/csrf"
)

type memSession struct {
	data map[string]any
}

func (s *memSession) Get(key string, fallback ...any) any {
	if v, ok := s.data[key]; ok {
		return v
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return nil
}
func (s *memSession) Put(key string, value any) { s.data[key] = value }
func (s *memSession) Flash(key string, value any) {
	s.data[key] = value
}
func (s *memSession) Pull(key string, fallback ...any) any {
	v := s.Get(key, fallback...)
	delete(s.data, key)
	return v
}
func (s *memSession) Forget(key string) { delete(s.data, key) }
func (s *memSession) Regenerate() error { return nil }
func (s *memSession) ID() string        { return "test" }

func newCSRFRequest(method, target string) (*http.Request, *memSession) {
	raw := httptest.NewRequest(method, target, nil)
	req := http.NewRequest(raw)
	if strings.HasPrefix(target, "https://") {
		req.Set("_forwarded_proto", "https")
	}
	sess := &memSession{data: map[string]any{}}
	req.SetSession(sess)
	return req, sess
}

func seedToken(req *http.Request) string {
	return csrf.Token(req)
}

func TestCSRFValidTokenSameOrigin(t *testing.T) {
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodPost, "https://app.example/form")
	token := seedToken(req)
	req.Raw().Header.Set("Origin", "https://app.example")
	req.Raw().Header.Set("X-CSRF-TOKEN", token)
	resp := handler(req)
	if resp.StatusCode() != 200 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
}

func TestCSRFInvalidToken(t *testing.T) {
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodPost, "https://app.example/form")
	_ = seedToken(req)
	req.Raw().Header.Set("Origin", "https://app.example")
	req.Raw().Header.Set("X-CSRF-TOKEN", "wrong")
	resp := handler(req)
	if resp.StatusCode() != 403 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
}

func TestCSRFMissingToken(t *testing.T) {
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodPost, "https://app.example/form")
	_ = seedToken(req)
	req.Raw().Header.Set("Origin", "https://app.example")
	resp := handler(req)
	if resp.StatusCode() != 403 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
}

func TestCSRFOriginValidation(t *testing.T) {
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodPost, "https://app.example/form")
	token := seedToken(req)
	req.Raw().Header.Set("Origin", "https://evil.example")
	req.Raw().Header.Set("X-CSRF-TOKEN", token)
	resp := handler(req)
	if resp.StatusCode() != 403 {
		t.Fatalf("status=%d want 403 for cross-origin", resp.StatusCode())
	}
	if !strings.Contains(string(resp.Content()), "origin") {
		t.Fatalf("body=%q", resp.Content())
	}
}

func TestCSRFCrossSiteRequest(t *testing.T) {
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodPost, "https://app.example/form")
	token := seedToken(req)
	req.Raw().Header.Set("Sec-Fetch-Site", "cross-site")
	req.Raw().Header.Set("X-CSRF-TOKEN", token)
	resp := handler(req)
	if resp.StatusCode() != 403 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
}

func TestCSRFSameSiteSecFetchAllowed(t *testing.T) {
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodPost, "https://app.example/form")
	token := seedToken(req)
	req.Raw().Header.Set("Sec-Fetch-Site", "same-origin")
	req.Raw().Header.Set("X-CSRF-TOKEN", token)
	resp := handler(req)
	if resp.StatusCode() != 200 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
}

func TestCSRFGETSkipsVerification(t *testing.T) {
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodGet, "https://app.example/")
	resp := handler(req)
	if resp.StatusCode() != 200 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
}

func TestCSRFPUTPATCHDELETERequireToken(t *testing.T) {
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	for _, method := range []string{stdhttp.MethodPut, stdhttp.MethodPatch, stdhttp.MethodDelete} {
		req, _ := newCSRFRequest(method, "https://app.example/resource")
		_ = seedToken(req)
		req.Raw().Header.Set("Origin", "https://app.example")
		resp := handler(req)
		if resp.StatusCode() != 403 {
			t.Fatalf("%s without token status=%d", method, resp.StatusCode())
		}
	}
}

func TestCSRFNullOriginBlocked(t *testing.T) {
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodPost, "https://app.example/form")
	token := seedToken(req)
	req.Raw().Header.Set("Origin", "null")
	req.Raw().Header.Set("X-CSRF-TOKEN", token)
	resp := handler(req)
	if resp.StatusCode() != 403 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
}

func TestCSRFNoOriginTokenOnly(t *testing.T) {
	// Non-browser clients without Origin still authenticate via token.
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodPost, "https://app.example/api")
	token := seedToken(req)
	req.Raw().Header.Set("X-CSRF-TOKEN", token)
	resp := handler(req)
	if resp.StatusCode() != 200 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
}

func xsrfCookie(t *testing.T, resp *http.Response) *stdhttp.Cookie {
	t.Helper()
	for _, c := range resp.Cookies() {
		if c != nil && c.Name == "XSRF-TOKEN" {
			return c
		}
	}
	t.Fatal("missing XSRF-TOKEN")
	return nil
}

func TestXSRFCookieSecureInProduction(t *testing.T) {
	cookie.SetProductionPolicy(true)
	t.Cleanup(func() { cookie.SetProductionPolicy(false) })
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("SESSION_SECURE", "")
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodGet, "http://app.example/form")
	resp := handler(req)
	c := xsrfCookie(t, resp)
	if !c.Secure {
		t.Fatal("production XSRF-TOKEN must be Secure")
	}
	if c.HttpOnly {
		t.Fatal("XSRF-TOKEN HttpOnly must stay false")
	}
	if c.SameSite != stdhttp.SameSiteLaxMode {
		t.Fatalf("SameSite=%v", c.SameSite)
	}
}

func TestXSRFCookieSecureOnHTTPSInDevelopment(t *testing.T) {
	cookie.SetProductionPolicy(false)
	t.Setenv("COOKIE_SECURE", "")
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodGet, "https://app.example/form")
	resp := handler(req)
	c := xsrfCookie(t, resp)
	if !c.Secure {
		t.Fatal("HTTPS development XSRF-TOKEN must be Secure")
	}
	if c.HttpOnly {
		t.Fatal("XSRF-TOKEN HttpOnly must stay false")
	}
}

func TestXSRFCookieNotForcedSecureOnHTTPInDevelopment(t *testing.T) {
	cookie.SetProductionPolicy(false)
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("SESSION_SECURE", "")
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodGet, "http://app.example/form")
	resp := handler(req)
	c := xsrfCookie(t, resp)
	if c.Secure {
		t.Fatal("HTTP development XSRF-TOKEN must not force Secure")
	}
}

func hasResponseCookie(resp *http.Response, name string) bool {
	for _, c := range resp.Cookies() {
		if c != nil && c.Name == name {
			return true
		}
	}
	return false
}

func TestCSRFSkipAnonymousSeedNoSessionCookie(t *testing.T) {
	csrf.SkipAnonymousSeed(func(req *http.Request) bool {
		return req.Path() == "/"
	})
	t.Cleanup(func() { csrf.SkipAnonymousSeed(nil) })

	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, sess := newCSRFRequest(stdhttp.MethodGet, "https://app.example/")
	resp := handler(req)
	if resp.StatusCode() != 200 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
	if hasResponseCookie(resp, "XSRF-TOKEN") {
		t.Fatal("expected no XSRF-TOKEN cookie for anonymous public GET")
	}
	if sess.Get("_csrf_token") != nil {
		t.Fatal("expected no _csrf_token in session")
	}
}

func TestCSRFSkipAnonymousSeedWithSessionCookie(t *testing.T) {
	csrf.SkipAnonymousSeed(func(req *http.Request) bool {
		return req.Path() == "/"
	})
	t.Cleanup(func() { csrf.SkipAnonymousSeed(nil) })

	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, sess := newCSRFRequest(stdhttp.MethodGet, "https://app.example/")
	req.Raw().AddCookie(&stdhttp.Cookie{Name: csrf.DefaultSessionCookie, Value: "aabbccddeeff00112233445566778899"})
	resp := handler(req)
	if resp.StatusCode() != 200 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
	if !hasResponseCookie(resp, "XSRF-TOKEN") {
		t.Fatal("expected XSRF-TOKEN cookie when session cookie present")
	}
	if tok, _ := sess.Get("_csrf_token").(string); tok == "" {
		t.Fatal("expected _csrf_token seeded in session")
	}
}

func exceptAPI() func(*http.Request) *http.Response {
	return csrf.Except("/api")(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
}

func postExcept(t *testing.T, target string) *http.Response {
	t.Helper()
	req, _ := newCSRFRequest(stdhttp.MethodPost, target)
	_ = seedToken(req)
	req.Raw().Header.Set("Origin", "https://app.example")
	return exceptAPI()(req)
}

func TestCSRFExceptAPIBoundary(t *testing.T) {
	allowed := []string{
		"https://app.example/api",
		"https://app.example/api/",
		"https://app.example/api/users",
		"https://app.example/api/users/1",
		"https://app.example/api?x=1",
	}
	for _, target := range allowed {
		resp := postExcept(t, target)
		if resp.StatusCode() != 200 {
			t.Fatalf("%s: status=%d want except 200", target, resp.StatusCode())
		}
	}

	blocked := []string{
		"https://app.example/apitoken",
		"https://app.example/api-v2",
		"https://app.example/api2",
		"https://app.example/api/../secret",
		"https://app.example/other/api",
	}
	for _, target := range blocked {
		resp := postExcept(t, target)
		if resp.StatusCode() != 403 {
			t.Fatalf("%s: status=%d want CSRF 403", target, resp.StatusCode())
		}
	}
}

func TestCSRFExceptEncodedApitoken(t *testing.T) {
	resp := postExcept(t, "https://app.example/api%74oken")
	if resp.StatusCode() != 403 {
		t.Fatalf("encoded /apitoken must not be excepted: status=%d", resp.StatusCode())
	}
}

func TestCSRFExceptDoesNotUseQueryAsPath(t *testing.T) {
	resp := postExcept(t, "https://app.example/form?path=/api")
	if resp.StatusCode() != 403 {
		t.Fatalf("query must not except: status=%d", resp.StatusCode())
	}
}

func TestCSRFSeesMethodOverrideDELETE(t *testing.T) {
	handler := csrf.Middleware(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodPost, "https://app.example/form")
	_ = seedToken(req)
	req.Raw().Header.Set("Origin", "https://app.example")
	req.Raw().Header.Set("X-HTTP-Method-Override", "DELETE")
	middleware.ApplyMethodOverride(req)
	resp := handler(req)
	if resp.StatusCode() != 403 {
		t.Fatalf("overridden DELETE must require CSRF: status=%d", resp.StatusCode())
	}
}

func TestCSRFExceptAPIStillBypassesOverriddenDELETE(t *testing.T) {
	req, _ := newCSRFRequest(stdhttp.MethodPost, "https://app.example/api/users")
	_ = seedToken(req)
	req.Raw().Header.Set("Origin", "https://app.example")
	req.Raw().Header.Set("X-HTTP-Method-Override", "DELETE")
	middleware.ApplyMethodOverride(req)
	resp := exceptAPI()(req)
	if resp.StatusCode() != 200 {
		t.Fatalf("API except must still bypass overridden DELETE: status=%d", resp.StatusCode())
	}
}
