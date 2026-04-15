package config

import "time"

// Security configures session (Redis), CSRF (HTML forms), and JWT (API / SPA / mobile).
type Security struct {
	SessionEnabled bool `mapstructure:"session_enabled"`
	CSRFEnabled    bool `mapstructure:"csrf_enabled"`
	// CSRFSkipPrefixes skips CSRF for paths (e.g. /api/ for Bearer JWT).
	CSRFSkipPrefixes []string `mapstructure:"csrf_skip_prefixes"`
	TrustedOrigins   []string `mapstructure:"trusted_origins"`

	JWTSecret string        `mapstructure:"jwt_secret"`
	JWTIssuer string        `mapstructure:"jwt_issuer"`
	JWTExpiry time.Duration `mapstructure:"jwt_expiry"`

	// CookieSecure sets Secure flag on session and CSRF cookies (enable in production behind HTTPS).
	CookieSecure bool `mapstructure:"cookie_secure"`
	// DemoTokenEndpoint enables POST /api/v1/auth/token for local JWT testing only (never in prod).
	DemoTokenEndpoint bool `mapstructure:"demo_token_endpoint"`
}

