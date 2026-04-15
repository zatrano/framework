package config

import (
	"fmt"
	"strings"
	"time"
)

// HTTP bundles optional Fiber middleware (CORS, rate limit, per-request timeout, body size).
type HTTP struct {
	CORSEnabled          bool     `mapstructure:"cors_enabled"`
	CORSAllowOrigins     []string `mapstructure:"cors_allow_origins"`
	CORSAllowMethods     []string `mapstructure:"cors_allow_methods"`
	CORSAllowHeaders     []string `mapstructure:"cors_allow_headers"`
	CORSExposeHeaders    []string `mapstructure:"cors_expose_headers"`
	CORSAllowCredentials bool     `mapstructure:"cors_allow_credentials"`
	CORSMaxAge           int      `mapstructure:"cors_max_age"`

	RateLimitEnabled bool          `mapstructure:"rate_limit_enabled"`
	RateLimitMax     int           `mapstructure:"rate_limit_max"`
	RateLimitWindow  time.Duration `mapstructure:"rate_limit_window"`
	RateLimitRedis   bool          `mapstructure:"rate_limit_redis"`

	// RequestTimeout caps handler work per request (0 = disabled; see also http_read_timeout on the listener).
	RequestTimeout time.Duration `mapstructure:"request_timeout"`
	// BodyLimit is max request body bytes (0 = Fiber default, 4 MiB).
	BodyLimit int `mapstructure:"body_limit"`
}

func (c *Config) validateHTTP() error {
	h := c.HTTP
	if h.CORSEnabled && h.CORSAllowCredentials {
		for _, o := range h.CORSAllowOrigins {
			if strings.TrimSpace(o) == "*" {
				return fmt.Errorf("http.cors_allow_credentials cannot be used with http.cors_allow_origins containing \"*\" (browser security)")
			}
		}
	}
	if h.RateLimitEnabled {
		if h.RateLimitMax <= 0 {
			return fmt.Errorf("http.rate_limit_max must be > 0 when http.rate_limit_enabled is true")
		}
		if h.RateLimitWindow <= 0 {
			return fmt.Errorf("http.rate_limit_window must be > 0 when http.rate_limit_enabled is true")
		}
	}
	if h.RateLimitRedis && strings.TrimSpace(c.RedisURL) == "" {
		return fmt.Errorf("http.rate_limit_redis requires redis_url")
	}
	if h.BodyLimit < 0 {
		return fmt.Errorf("http.body_limit cannot be negative")
	}
	return nil
}

func (c *Config) applyHTTPDefaults() {
	h := &c.HTTP
	if h.CORSEnabled && len(h.CORSAllowOrigins) == 0 {
		h.CORSAllowOrigins = []string{"*"}
	}
	if h.RateLimitEnabled && h.RateLimitMax <= 0 {
		h.RateLimitMax = 100
	}
	if h.RateLimitEnabled && h.RateLimitWindow <= 0 {
		h.RateLimitWindow = time.Minute
	}
}

