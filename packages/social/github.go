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

// GitHub builds a real GitHub OAuth provider (falls back to stub when credentials are placeholders).
// In production stub fallback is disabled (see SetAllowStubProviders).
func GitHub(cfg Config) Provider {
	if cfg.ClientID == "" {
		cfg.ClientID = "github-client-id"
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"read:user", "user:email"}
	}
	if isPlaceholderOAuth(cfg.ClientID, cfg.ClientSecret) {
		if !allowStubProviders {
			return &disabledProvider{name: "github", err: ErrStubNotAllowedInProduction}
		}
		return NewStubProvider("github", cfg)
	}
	return &githubProvider{cfg: cfg, httpClient: &http.Client{Timeout: 15 * time.Second}}
}

type githubProvider struct {
	cfg        Config
	httpClient *http.Client
}

func (p *githubProvider) Name() string { return "github" }

func (p *githubProvider) RedirectURL(state string) string {
	values := url.Values{}
	values.Set("client_id", p.cfg.ClientID)
	values.Set("redirect_uri", p.cfg.RedirectURL)
	values.Set("scope", strings.Join(p.cfg.Scopes, " "))
	values.Set("state", state)
	values.Set("allow_signup", "true")
	return "https://github.com/login/oauth/authorize?" + values.Encode()
}

func (p *githubProvider) UserFromCode(code string) (*User, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}
	token, err := p.exchangeCode(code)
	if err != nil {
		return nil, err
	}
	info, err := p.fetchUser(token)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(fmt.Sprint(info["id"]))
	login := strings.TrimSpace(fmt.Sprint(info["login"]))
	name := strings.TrimSpace(fmt.Sprint(info["name"]))
	if name == "" || name == "<nil>" {
		name = login
	}
	if id == "" || id == "<nil>" {
		return nil, fmt.Errorf("github user id missing")
	}

	email, verified, err := p.resolveVerifiedEmail(token)
	if err != nil {
		return nil, err
	}
	return &User{
		ID:            id,
		Nickname:      login,
		Name:          name,
		Email:         email,
		Avatar:        strings.TrimSpace(fmt.Sprint(info["avatar_url"])),
		Provider:      "github",
		Token:         token,
		EmailVerified: verified,
		Raw:           info,
	}, nil
}

func (p *githubProvider) resolveVerifiedEmail(accessToken string) (email string, verified bool, err error) {
	email, verified, err = p.fetchVerifiedEmail(accessToken)
	if err != nil {
		return "", false, err
	}
	if email != "" {
		return email, verified, nil
	}
	// Profile email alone is not trusted without the verified emails API result.
	return "", false, fmt.Errorf("github verified email missing (grant user:email scope)")
}

func (p *githubProvider) exchangeCode(code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", p.cfg.ClientID)
	form.Set("client_secret", p.cfg.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", p.cfg.RedirectURL)

	req, err := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	res, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("github token exchange failed: %s", strings.TrimSpace(string(body)))
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if errDesc := strings.TrimSpace(fmt.Sprint(payload["error"])); errDesc != "" && errDesc != "<nil>" {
		return "", fmt.Errorf("github token exchange failed: %s", errDesc)
	}
	token := strings.TrimSpace(fmt.Sprint(payload["access_token"]))
	if token == "" || token == "<nil>" {
		return "", fmt.Errorf("github access_token missing")
	}
	return token, nil
}

func (p *githubProvider) fetchUser(accessToken string) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("github user failed: %s", strings.TrimSpace(string(body)))
	}
	var info map[string]any
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return info, nil
}

func (p *githubProvider) fetchVerifiedEmail(accessToken string) (string, bool, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := p.httpClient.Do(req)
	if err != nil {
		return "", false, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return "", false, fmt.Errorf("github emails failed: %s", strings.TrimSpace(string(body)))
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		return "", false, err
	}
	var fallback string
	for _, row := range rows {
		email := strings.TrimSpace(fmt.Sprint(row["email"]))
		if email == "" || email == "<nil>" {
			continue
		}
		verified, present := parseEmailVerified(row["verified"])
		if !present || !verified {
			continue
		}
		if primary, _ := row["primary"].(bool); primary {
			return email, true, nil
		}
		if fallback == "" {
			fallback = email
		}
	}
	if fallback != "" {
		return fallback, true, nil
	}
	return "", false, nil
}
