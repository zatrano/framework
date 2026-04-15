package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config is the runtime configuration for ZATRANO (HTTP, data stores, logging, security).
// Values are loaded from optional .env, config/{env}.yaml, and ZATRANO_* environment variables
// (nested YAML keys map to env like ZATRANO_SECURITY_JWT_SECRET).
type Config struct {
	Env     string `mapstructure:"env"`
	AppName string `mapstructure:"app_name"`

	HTTPAddr        string        `mapstructure:"http_addr"`
	HTTPReadTimeout time.Duration `mapstructure:"http_read_timeout"`

	DatabaseURL      string `mapstructure:"database_url"`
	DatabaseRequired bool   `mapstructure:"database_required"`

	RedisURL      string `mapstructure:"redis_url"`
	RedisRequired bool   `mapstructure:"redis_required"`

	LogLevel       string `mapstructure:"log_level"`
	LogDevelopment bool   `mapstructure:"log_development"`

	// StaticPath is the local directory for public assets (optional).
	StaticPath string `mapstructure:"static_path"`
	// StaticURLPrefix is the URL prefix for static files (e.g. /static).
	StaticURLPrefix string `mapstructure:"static_url_prefix"`

	MigrationsDir string `mapstructure:"migrations_dir"`
	SeedsDir      string `mapstructure:"seeds_dir"`
	OpenAPIPath   string `mapstructure:"openapi_path"`

	Security Security `mapstructure:"security"`
	OAuth    OAuth    `mapstructure:"oauth"`
	HTTP     HTTP     `mapstructure:"http"`
	I18n     I18n     `mapstructure:"i18n"`
}

// LoadOptions controls where configuration is read from.
type LoadOptions struct {
	// Env is the profile name (e.g. dev, prod). Defaults to ZATRANO_ENV or "dev".
	Env string
	// ConfigDir is the directory containing {env}.yaml (default "config").
	ConfigDir string
	// DotEnv, if true, loads .env from the working directory when present.
	DotEnv bool
}

// Load reads configuration and returns a validated Config.
func Load(opts LoadOptions) (*Config, error) {
	if opts.ConfigDir == "" {
		opts.ConfigDir = "config"
	}
	envName := strings.TrimSpace(opts.Env)
	if envName == "" {
		envName = strings.TrimSpace(os.Getenv("ZATRANO_ENV"))
	}
	if envName == "" {
		envName = "dev"
	}

	if opts.DotEnv {
		_ = godotenv.Load()
	}

	v := viper.New()
	v.SetEnvPrefix("ZATRANO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetConfigName(envName)
	v.SetConfigType("yaml")
	v.AddConfigPath(opts.ConfigDir)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			v.Set("env", envName)
		} else {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	// Defaults when file is missing or keys omitted
	v.SetDefault("env", envName)
	v.SetDefault("app_name", "ZATRANO")
	v.SetDefault("http_addr", ":8080")
	v.SetDefault("http_read_timeout", 30*time.Second)
	v.SetDefault("database_required", false)
	v.SetDefault("redis_required", false)
	v.SetDefault("log_level", "info")
	v.SetDefault("log_development", envName == "dev")
	v.SetDefault("static_path", "public")
	v.SetDefault("static_url_prefix", "/static")
	v.SetDefault("migrations_dir", "migrations")
	v.SetDefault("seeds_dir", "db/seeds")
	v.SetDefault("openapi_path", "api/openapi.yaml")
	v.SetDefault("security.session_enabled", true)
	v.SetDefault("security.csrf_enabled", true)
	v.SetDefault("security.jwt_issuer", "zatrano")
	v.SetDefault("security.jwt_expiry", 60*time.Minute)
	v.SetDefault("security.demo_token_endpoint", false)
	v.SetDefault("oauth.enabled", false)
	v.SetDefault("http.cors_enabled", false)
	v.SetDefault("http.rate_limit_enabled", false)
	v.SetDefault("http.rate_limit_max", 100)
	v.SetDefault("http.rate_limit_window", time.Minute)
	v.SetDefault("http.rate_limit_redis", false)
	v.SetDefault("http.request_timeout", time.Duration(0))
	v.SetDefault("http.body_limit", 0)
	v.SetDefault("i18n.enabled", false)
	v.SetDefault("i18n.default_locale", "en")
	v.SetDefault("i18n.locales_dir", "locales")
	v.SetDefault("i18n.cookie_name", "zatrano_lang")
	v.SetDefault("i18n.query_key", "lang")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.applyDerivedDefaults()

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	switch strings.ToLower(strings.TrimSpace(c.LogLevel)) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid log_level %q (use debug, info, warn, error)", c.LogLevel)
	}
	if c.DatabaseRequired && strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("database_required is true but database_url is empty (set ZATRANO_DATABASE_URL or config/database_url)")
	}
	if c.RedisRequired && strings.TrimSpace(c.RedisURL) == "" {
		return fmt.Errorf("redis_required is true but redis_url is empty (set ZATRANO_REDIS_URL or config/redis_url)")
	}
	if c.Security.DemoTokenEndpoint && strings.EqualFold(strings.TrimSpace(c.Env), "prod") {
		return fmt.Errorf("security.demo_token_endpoint cannot be true when env is prod")
	}
	if c.OAuth.Enabled {
		if strings.TrimSpace(c.RedisURL) == "" {
			return fmt.Errorf("oauth.enabled requires redis_url (state storage)")
		}
		if strings.TrimSpace(c.OAuth.BaseURL) == "" {
			return fmt.Errorf("oauth.base_url is required when oauth.enabled is true")
		}
		if !oauthProviderConfigured(c.OAuth.Providers.Google) && !oauthProviderConfigured(c.OAuth.Providers.Github) {
			return fmt.Errorf("oauth.enabled requires at least one provider with client_id (google or github)")
		}
	}
	if err := c.validateHTTP(); err != nil {
		return err
	}
	if err := c.validateI18n(); err != nil {
		return err
	}
	return nil
}

func oauthProviderConfigured(p OAuthProvider) bool {
	return strings.TrimSpace(p.ClientID) != "" && strings.TrimSpace(p.ClientSecret) != ""
}

func (c *Config) applyDerivedDefaults() {
	if strings.TrimSpace(c.RedisURL) == "" {
		c.Security.SessionEnabled = false
		c.Security.CSRFEnabled = false
	}
	if len(c.Security.CSRFSkipPrefixes) == 0 {
		c.Security.CSRFSkipPrefixes = []string{"/api/"}
	}
	if c.Security.JWTExpiry <= 0 {
		c.Security.JWTExpiry = 60 * time.Minute
	}
	if strings.TrimSpace(c.Security.JWTIssuer) == "" {
		c.Security.JWTIssuer = "zatrano"
	}
	if strings.TrimSpace(c.MigrationsDir) == "" {
		c.MigrationsDir = "migrations"
	}
	if strings.TrimSpace(c.SeedsDir) == "" {
		c.SeedsDir = "db/seeds"
	}
	if strings.TrimSpace(c.OpenAPIPath) == "" {
		c.OpenAPIPath = "api/openapi.yaml"
	}
	c.Security.CSRFSkipPrefixes = appendUniquePrefix(c.Security.CSRFSkipPrefixes, "/auth/oauth/")
	c.applyHTTPDefaults()
	c.applyI18nDefaults()
}

func appendUniquePrefix(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

