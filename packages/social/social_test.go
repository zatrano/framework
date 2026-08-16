package social_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/social"
)

func TestSocialRedirectAndUser(t *testing.T) {
	m := social.New()
	m.Extend("github", social.GitHub(social.Config{
		ClientID:    "id",
		RedirectURL: "http://localhost/callback",
	}))
	url, state, err := m.Redirect("github")
	if err != nil || state == "" || !strings.Contains(url, "authorize") {
		t.Fatalf("redirect failed url=%q state=%q err=%v", url, state, err)
	}
	if strings.Contains(url, "oauth.zatrano.test") {
		t.Fatalf("stub must not use oauth subdomain: %s", url)
	}
	if !strings.Contains(url, "http://localhost/oauth/github/authorize") {
		t.Fatalf("expected same-origin stub authorize, got %s", url)
	}
	if !m.ValidateState(state) {
		t.Fatal("expected valid state")
	}
	if m.ValidateState(state) {
		t.Fatal("state should be single-use")
	}
	user, err := m.User("github", "demo")
	if err != nil || user.Email == "" || user.Provider != "github" {
		t.Fatalf("user=%#v err=%v", user, err)
	}
}

func TestStubAuthorizeSameOrigin(t *testing.T) {
	p := social.NewStubProvider("google", social.Config{
		ClientID:    "google-client-id",
		RedirectURL: "http://localhost:8080/auth/google/callback",
		Scopes:      []string{"openid", "profile", "email"},
	})
	authURL := p.RedirectURL("state-1")
	if strings.Contains(authURL, "oauth.zatrano.test") {
		t.Fatalf("stub must not use oauth subdomain: %s", authURL)
	}
	if !strings.HasPrefix(authURL, "http://localhost:8080/oauth/google/authorize?") {
		t.Fatalf("expected same-origin authorize, got %s", authURL)
	}

	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, authURL, nil))
	resp := p.AuthorizeHandler()(req)
	if resp.StatusCode() != 302 {
		t.Fatalf("status=%d", resp.StatusCode())
	}
	loc := resp.RedirectURL()
	if !strings.HasPrefix(loc, "http://localhost:8080/auth/google/callback?") {
		t.Fatalf("callback=%s", loc)
	}
	if !strings.Contains(loc, "code=demo") || !strings.Contains(loc, "state=state-1") {
		t.Fatalf("callback missing code/state: %s", loc)
	}

	bad := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/oauth/google/authorize?redirect_uri=https://evil.test/cb&state=x", nil))
	denied := p.AuthorizeHandler()(bad)
	if denied.StatusCode() != 400 {
		t.Fatalf("expected 400 for foreign redirect, got %d", denied.StatusCode())
	}
}
