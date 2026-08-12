package oauth_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/oauth"
)

func TestOAuthClientCredentials(t *testing.T) {
	s := oauth.New()
	client := s.RegisterClient(oauth.Client{
		Name:         "Demo",
		RedirectURIs: []string{"http://localhost/callback"},
		Scopes:       []string{"read", "write"},
	})
	token, err := s.Token("client_credentials", client.ID, client.Secret, nil)
	if err != nil || token.Token == "" {
		t.Fatalf("token=%v err=%v", token, err)
	}
	info := s.Introspect(token.Token)
	if info["active"] != true {
		t.Fatalf("introspect=%v", info)
	}
}

func TestOAuthAuthorizationCodeWithPKCE(t *testing.T) {
	s := oauth.New()
	client := s.RegisterClient(oauth.Client{
		Name:         "Web",
		RedirectURIs: []string{"http://localhost/callback"},
	})
	verifier, challenge := oauth.PKCEChallengeS256()
	code, err := s.AuthorizeWith(oauth.AuthorizeParams{
		ClientID:            client.ID,
		RedirectURI:         "http://localhost/callback",
		UserID:              "42",
		Scope:               "read",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
	})
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.Token("authorization_code", client.ID, client.Secret, map[string]string{
		"code":          code,
		"redirect_uri":  "http://localhost/callback",
		"code_verifier": verifier,
	})
	if err != nil || token.UserID != "42" {
		t.Fatalf("token=%v err=%v", token, err)
	}
	if token.RefreshToken == "" {
		t.Fatal("expected refresh_token")
	}

	refreshed, err := s.Token("refresh_token", client.ID, client.Secret, map[string]string{
		"refresh_token": token.RefreshToken,
	})
	if err != nil || refreshed.Token == "" || refreshed.UserID != "42" {
		t.Fatalf("refresh=%v err=%v", refreshed, err)
	}
	if refreshed.RefreshToken == "" {
		t.Fatal("expected rotated refresh_token")
	}
}

func TestOAuthPKCERejectsBadVerifier(t *testing.T) {
	s := oauth.New()
	client := s.RegisterClient(oauth.Client{
		Name:         "Web",
		RedirectURIs: []string{"http://localhost/callback"},
	})
	_, challenge := oauth.PKCEChallengeS256()
	code, err := s.AuthorizeWith(oauth.AuthorizeParams{
		ClientID:            client.ID,
		RedirectURI:         "http://localhost/callback",
		UserID:              "1",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Token("authorization_code", client.ID, client.Secret, map[string]string{
		"code":          code,
		"redirect_uri":  "http://localhost/callback",
		"code_verifier": "not-the-verifier",
	})
	if err == nil {
		t.Fatal("expected PKCE failure")
	}
}

func TestOAuthAuthorizeRequiresChallenge(t *testing.T) {
	s := oauth.New()
	client := s.RegisterClient(oauth.Client{
		Name:         "Web",
		RedirectURIs: []string{"http://localhost/callback"},
	})
	_, err := s.Authorize(client.ID, "http://localhost/callback", "1", "read")
	if err == nil {
		t.Fatal("expected code_challenge required")
	}
}

func TestOAuthJSONStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "oauth.json")
	s, err := oauth.NewWithStore(path)
	if err != nil {
		t.Fatal(err)
	}
	client := s.RegisterClient(oauth.Client{
		Name:         "Persisted",
		RedirectURIs: []string{"http://localhost/callback"},
	})
	verifier, challenge := oauth.PKCEChallengeS256()
	code, err := s.AuthorizeWith(oauth.AuthorizeParams{
		ClientID:            client.ID,
		RedirectURI:         "http://localhost/callback",
		UserID:              "9",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
	})
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.Token("authorization_code", client.ID, client.Secret, map[string]string{
		"code":          code,
		"redirect_uri":  "http://localhost/callback",
		"code_verifier": verifier,
	})
	if err != nil {
		t.Fatal(err)
	}

	s2, err := oauth.NewWithStore(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s2.Client(client.ID)
	if err != nil || got.Name != "Persisted" {
		t.Fatalf("client=%v err=%v", got, err)
	}
	info := s2.Introspect(token.Token)
	if info["active"] != true {
		t.Fatalf("introspect after reload=%v", info)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
