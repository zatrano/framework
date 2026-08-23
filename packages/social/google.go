package social

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Google builds a real Google OAuth 2.0 provider (falls back to stub when credentials are placeholders).
// In production stub fallback is disabled (see SetAllowStubProviders).
func Google(cfg Config) Provider {
	if cfg.ClientID == "" {
		cfg.ClientID = "google-client-id"
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid", "profile", "email"}
	}
	if isPlaceholderOAuth(cfg.ClientID, cfg.ClientSecret) {
		if !allowStubProviders {
			return &disabledProvider{name: "google", err: ErrStubNotAllowedInProduction}
		}
		return NewStubProvider("google", cfg)
	}
	return &googleProvider{cfg: cfg, httpClient: &http.Client{Timeout: 15 * time.Second}}
}

func isPlaceholderOAuth(clientID, clientSecret string) bool {
	id := strings.TrimSpace(clientID)
	secret := strings.TrimSpace(clientSecret)
	if id == "" || secret == "" {
		return true
	}
	placeholders := []string{"google-client-id", "github-client-id", "google-client-secret", "github-client-secret"}
	for _, p := range placeholders {
		if strings.EqualFold(id, p) || strings.EqualFold(secret, p) {
			return true
		}
	}
	return false
}

type googleProvider struct {
	cfg        Config
	httpClient *http.Client
}

type disabledProvider struct {
	name string
	err  error
}

func (p *disabledProvider) Name() string { return p.name }

func (p *disabledProvider) RedirectURL(string) string { return "" }

func (p *disabledProvider) UserFromCode(string) (*User, error) {
	return nil, p.err
}

func (p *googleProvider) Name() string { return "google" }

func (p *googleProvider) RedirectURL(state string) string {
	values := url.Values{}
	values.Set("client_id", p.cfg.ClientID)
	values.Set("redirect_uri", p.cfg.RedirectURL)
	values.Set("response_type", "code")
	values.Set("scope", strings.Join(p.cfg.Scopes, " "))
	values.Set("state", state)
	values.Set("access_type", "online")
	values.Set("prompt", "select_account")
	return "https://accounts.google.com/o/oauth2/v2/auth?" + values.Encode()
}

func (p *googleProvider) UserFromCode(code string) (*User, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}
	token, err := p.exchangeCode(code)
	if err != nil {
		return nil, err
	}
	info, err := p.fetchUserInfo(token)
	if err != nil {
		return nil, err
	}
	return userFromGoogleInfo(info, token)
}

// userFromGoogleInfo maps OpenID userinfo and enforces email_verified fail-closed.
func userFromGoogleInfo(info map[string]any, token string) (*User, error) {
	email := strings.TrimSpace(fmt.Sprint(info["email"]))
	if email == "" || email == "<nil>" {
		return nil, fmt.Errorf("google email missing")
	}
	verified, present := parseEmailVerified(info["email_verified"])
	if !present {
		return nil, fmt.Errorf("google email_verified claim missing")
	}
	if !verified {
		return nil, fmt.Errorf("google email is not verified")
	}
	id := strings.TrimSpace(fmt.Sprint(info["sub"]))
	if id == "" || id == "<nil>" {
		return nil, fmt.Errorf("google sub missing")
	}
	name := strings.TrimSpace(fmt.Sprint(info["name"]))
	if name == "" || name == "<nil>" {
		name = strings.TrimSpace(fmt.Sprint(info["given_name"]))
	}
	return &User{
		ID:            id,
		Nickname:      strings.TrimSpace(fmt.Sprint(info["given_name"])),
		Name:          name,
		Email:         email,
		Avatar:        strings.TrimSpace(fmt.Sprint(info["picture"])),
		Provider:      "google",
		Token:         token,
		EmailVerified: true,
		Raw:           info,
	}, nil
}

func (p *googleProvider) exchangeCode(code string) (string, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", p.cfg.ClientID)
	form.Set("client_secret", p.cfg.ClientSecret)
	form.Set("redirect_uri", p.cfg.RedirectURL)
	form.Set("grant_type", "authorization_code")

	req, err := http.NewRequest(http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("google token exchange failed: %s", strings.TrimSpace(string(body)))
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	token := strings.TrimSpace(fmt.Sprint(payload["access_token"]))
	if token == "" || token == "<nil>" {
		return "", fmt.Errorf("google access_token missing")
	}
	return token, nil
}

func (p *googleProvider) fetchUserInfo(accessToken string) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("google userinfo failed: %s", strings.TrimSpace(string(body)))
	}
	var info map[string]any
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return info, nil
}
