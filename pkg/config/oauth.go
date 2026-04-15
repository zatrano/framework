package config

// OAuth configures browser-based OAuth2 login (Google, GitHub). Requires Redis for CSRF state + session for user binding.
type OAuth struct {
	Enabled bool   `mapstructure:"enabled"`
	BaseURL string `mapstructure:"base_url"` // e.g. https://app.example.com (no trailing slash)

	Providers OAuthProviders `mapstructure:"providers"`
}

// OAuthProviders holds optional OAuth app credentials per provider.
type OAuthProviders struct {
	Google OAuthProvider `mapstructure:"google"`
	Github OAuthProvider `mapstructure:"github"`
}

// OAuthProvider is a single OAuth2 client registration.
type OAuthProvider struct {
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	Scopes       []string `mapstructure:"scopes"`
}

