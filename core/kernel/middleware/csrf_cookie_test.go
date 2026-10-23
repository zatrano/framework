package middleware

import (
	stdhttp "net/http"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
)

// The browser attaches the XSRF-TOKEN cookie to every request, including a
// forged cross-site one, so the cookie alone must never count as the token.
func TestCSRFCookieAloneIsNotAToken(t *testing.T) {
	handler := CSRF(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	for _, origin := range []string{"", "https://app.example"} {
		req, _ := newCSRFRequest(stdhttp.MethodPost, "https://app.example/transfer")
		token := seedToken(req)
		req.SetCookie("XSRF-TOKEN", token)
		if origin != "" {
			req.SetHeader("Origin", origin)
		}
		if resp := handler(req); resp.StatusCode() != 403 {
			t.Fatalf("origin %q: cookie-only POST got status %d, want 403", origin, resp.StatusCode())
		}
	}
}

func TestCSRFHeaderTokenStillAccepted(t *testing.T) {
	handler := CSRF(func(req *http.Request) *http.Response {
		return http.Text("ok")
	})
	req, _ := newCSRFRequest(stdhttp.MethodPost, "https://app.example/transfer")
	token := seedToken(req)
	req.SetCookie("XSRF-TOKEN", token)
	req.SetHeader("Origin", "https://app.example")
	req.SetHeader("X-XSRF-TOKEN", token)
	if resp := handler(req); resp.StatusCode() != 200 {
		t.Fatalf("X-XSRF-TOKEN header got status %d", resp.StatusCode())
	}
}

func TestCSRFInstalledFlag(t *testing.T) {
	_ = CSRFExcept("/api")(func(req *http.Request) *http.Response { return nil })
	if !CSRFInstalled() {
		t.Fatal("wrapping a handler with CSRFExcept did not set CSRFInstalled")
	}
}
