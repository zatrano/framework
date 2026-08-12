package social_test

import (
	"strings"
	"testing"

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
