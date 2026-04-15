package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"

	"github.com/zatrano/framework/pkg/core"
)

// googleOAuthEndpoint avoids importing golang.org/x/oauth2/google (extra cloud deps for DX).
var googleOAuthEndpoint = oauth2.Endpoint{
	AuthURL:   "https://accounts.google.com/o/oauth2/auth",
	TokenURL:  "https://oauth2.googleapis.com/token",
	AuthStyle: oauth2.AuthStyleInParams,
}

const stateKeyPrefix = "zatrano:oauth:state:"

// Register mounts Google/GitHub OAuth2 routes when enabled in config.
func Register(a *core.App, app *fiber.App) {
	cfg := a.Config.OAuth
	if !cfg.Enabled || a.Redis == nil {
		return
	}

	providers := map[string]oauth2.Config{}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")

	if p := cfg.Providers.Google; p.ClientID != "" && p.ClientSecret != "" {
		scopes := p.Scopes
		if len(scopes) == 0 {
			scopes = []string{"openid", "email", "profile"}
		}
		providers["google"] = oauth2.Config{
			ClientID:     p.ClientID,
			ClientSecret: p.ClientSecret,
			RedirectURL:  base + "/auth/oauth/google/callback",
			Scopes:       scopes,
			Endpoint:     googleOAuthEndpoint,
		}
	}
	if p := cfg.Providers.Github; p.ClientID != "" && p.ClientSecret != "" {
		scopes := p.Scopes
		if len(scopes) == 0 {
			scopes = []string{"read:user", "user:email"}
		}
		providers["github"] = oauth2.Config{
			ClientID:     p.ClientID,
			ClientSecret: p.ClientSecret,
			RedirectURL:  base + "/auth/oauth/github/callback",
			Scopes:       scopes,
			Endpoint:     githuboauth.Endpoint,
		}
	}
	if len(providers) == 0 {
		return
	}

	app.Get("/auth/oauth/:provider/login", func(c fiber.Ctx) error {
		p := strings.ToLower(strings.TrimSpace(c.Params("provider")))
		oauthCfg, ok := providers[p]
		if !ok {
			return fiber.NewError(fiber.StatusNotFound, "unknown oauth provider")
		}
		state, err := randomState()
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		ctx := c.Context()
		rkey := stateKeyPrefix + state
		if err := a.Redis.Set(ctx, rkey, p, 10*time.Minute).Err(); err != nil {
			a.Log.Error("oauth state store", zap.Error(err))
			return fiber.NewError(fiber.StatusInternalServerError, "oauth state failed")
		}
		url := oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
		return c.Redirect().To(url)
	})

	app.Get("/auth/oauth/:provider/callback", func(c fiber.Ctx) error {
		p := strings.ToLower(strings.TrimSpace(c.Params("provider")))
		oauthCfg, ok := providers[p]
		if !ok {
			return fiber.NewError(fiber.StatusNotFound, "unknown oauth provider")
		}
		q := c.Queries()
		state := q["state"]
		code := q["code"]
		if state == "" || code == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing code or state")
		}
		ctx := c.Context()
		rkey := stateKeyPrefix + state
		want, err := a.Redis.Get(ctx, rkey).Result()
		if err == redis.Nil || want != p {
			return fiber.NewError(fiber.StatusBadRequest, "invalid oauth state")
		}
		_ = a.Redis.Del(ctx, rkey)

		tok, err := oauthCfg.Exchange(ctx, code)
		if err != nil {
			a.Log.Warn("oauth exchange", zap.Error(err))
			return fiber.NewError(fiber.StatusBadRequest, "token exchange failed")
		}

		sub, name, email, err := fetchProfile(ctx, p, tok)
		if err != nil {
			a.Log.Warn("oauth profile", zap.Error(err))
			return fiber.NewError(fiber.StatusBadGateway, "profile fetch failed")
		}

		smw := session.FromContext(c)
		if smw == nil {
			return fiber.NewError(fiber.StatusInternalServerError, "session middleware missing")
		}
		smw.Set("oauth_provider", p)
		smw.Set("oauth_subject", sub)
		smw.Set("oauth_name", name)
		smw.Set("oauth_email", email)

		return c.Redirect().To("/")
	})
}

func randomState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func fetchProfile(ctx context.Context, provider string, tok *oauth2.Token) (sub, name, email string, err error) {
	hc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(tok))
	switch provider {
	case "google":
		return fetchGoogleProfile(hc)
	case "github":
		return fetchGitHubProfile(hc)
	default:
		return "", "", "", fmt.Errorf("unknown provider")
	}
}

func fetchGoogleProfile(hc *http.Client) (sub, name, email string, err error) {
	req, _ := http.NewRequest(http.MethodGet, "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	resp, err := hc.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", "", "", fmt.Errorf("google userinfo: %s: %s", resp.Status, string(b))
	}
	var u struct {
		Sub   string `json:"sub"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return "", "", "", err
	}
	return u.Sub, u.Name, u.Email, nil
}

func fetchGitHubProfile(hc *http.Client) (sub, name, email string, err error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := hc.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", "", "", fmt.Errorf("github user: %s: %s", resp.Status, string(b))
	}
	var u struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return "", "", "", err
	}
	sub = fmt.Sprintf("%d", u.ID)
	name = u.Name
	if name == "" {
		name = u.Login
	}
	email = u.Email
	if email == "" {
		email, err = fetchGitHubPrimaryEmail(hc)
		if err != nil {
			email = ""
		}
	}
	return sub, name, email, nil
}

func fetchGitHubPrimaryEmail(hc *http.Client) (string, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user/emails", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github emails: %s", resp.Status)
	}
	var rows []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return "", err
	}
	for _, r := range rows {
		if r.Primary {
			return r.Email, nil
		}
	}
	if len(rows) > 0 {
		return rows[0].Email, nil
	}
	return "", nil
}

// LoginURL returns an absolute login URL for a provider (for templates).
func LoginURL(base, provider string) string {
	return strings.TrimRight(base, "/") + "/auth/oauth/" + strings.Trim(provider, "/") + "/login"
}
