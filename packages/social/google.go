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
func Google(cfg Config) Provider {
	if cfg.ClientID == "" {
		cfg.ClientID = "google-client-id"
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid", "profile", "email"}
	}
	if isPlaceholderOAuth(cfg.ClientID, cfg.ClientSecret) {
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
	email := strings.TrimSpace(fmt.Sprint(info["email"]))
	name := strings.TrimSpace(fmt.Sprint(info["name"]))
	if name == "" {
		name = strings.TrimSpace(fmt.Sprint(info["given_name"]))
	}
	id := strings.TrimSpace(fmt.Sprint(info["sub"]))
	if id == "" {
		id = strings.TrimSpace(fmt.Sprint(info["id"]))
	}
	if email == "" || id == "" {
		return nil, fmt.Errorf("google userinfo incomplete")
	}
	return &User{
		ID:       id,
		Nickname: strings.TrimSpace(fmt.Sprint(info["given_name"])),
		Name:     name,
		Email:    email,
		Avatar:   strings.TrimSpace(fmt.Sprint(info["picture"])),
		Provider: "google",
		Token:    token,
		Raw:      info,
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
