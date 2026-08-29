package csrf_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/middleware/csrf"
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
