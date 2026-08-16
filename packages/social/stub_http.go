package social

import (
	"net/url"
	"strings"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// AuthorizeHandler completes the local stub flow on the same origin.
// GET /oauth/{provider}/authorize → redirect_uri?code=demo&state=...
func (p *StubProvider) AuthorizeHandler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		redirect := strings.TrimSpace(req.Query("redirect_uri"))
		if redirect == "" {
			redirect = strings.TrimSpace(p.cfg.RedirectURL)
		}
		if !p.allowedRedirect(redirect) {
			return http.JSON(map[string]any{
				"error":             "invalid_request",
				"error_description": "invalid redirect_uri",
			}).Status(400)
		}
		u, err := url.Parse(redirect)
		if err != nil {
			return http.JSON(map[string]any{"error": "invalid_request"}).Status(400)
		}
		q := u.Query()
		q.Set("code", "demo")
		if state := strings.TrimSpace(req.Query("state")); state != "" {
			q.Set("state", state)
		}
		u.RawQuery = q.Encode()
		return http.Redirect(u.String(), 302)
	}
}

func (p *StubProvider) allowedRedirect(redirect string) bool {
	want := strings.TrimSpace(p.cfg.RedirectURL)
	if want == "" || strings.TrimSpace(redirect) == "" {
		return false
	}
	return redirect == want
}
