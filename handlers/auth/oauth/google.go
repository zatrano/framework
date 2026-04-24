package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/zatrano/framework/configs/envconfig"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleProvider struct {
	config *oauth2.Config
}

func NewGoogleProvider() *GoogleProvider {
	redirectURI := strings.TrimSpace(envconfig.String("GOOGLE_REDIRECT_URI", ""))
	return &GoogleProvider{
		config: &oauth2.Config{
			ClientID:     envconfig.String("GOOGLE_CLIENT_ID", ""),
			ClientSecret: envconfig.String("GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:  redirectURI,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
	}
}

func (p *GoogleProvider) Name() string {
	return "google"
}

func (p *GoogleProvider) DisplayName() string {
	return "Google"
}

func (p *GoogleProvider) Config() *oauth2.Config {
	return p.config
}

func (p *GoogleProvider) LoginURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *GoogleProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.config.Exchange(ctx, code)
}

func (p *GoogleProvider) GetUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	client := p.config.Client(context.Background(), token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("google API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google API error: %s", resp.Status)
	}

	var googleUser struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("failed to decode google response: %w", err)
	}

	return &OAuthUserInfo{
		ProviderID: googleUser.ID,
		Email:      googleUser.Email,
		Name:       googleUser.Name,
		AvatarURL:  googleUser.Picture,
	}, nil
}
