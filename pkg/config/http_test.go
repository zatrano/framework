package config

import (
	"testing"
	"time"
)

func TestValidateHTTP_corsCredentialsWildcard(t *testing.T) {
	c := &Config{
		HTTP: HTTP{
			CORSEnabled:          true,
			CORSAllowOrigins:     []string{"*"},
			CORSAllowCredentials: true,
		},
	}
	if err := c.validateHTTP(); err == nil {
		t.Fatal("expected error for credentials + *")
	}
}

func TestValidateHTTP_rateLimitRedisWithoutURL(t *testing.T) {
	c := &Config{
		HTTP: HTTP{
			RateLimitEnabled: true,
			RateLimitMax:     10,
			RateLimitWindow:  time.Minute,
			RateLimitRedis:   true,
		},
	}
	if err := c.validateHTTP(); err == nil {
		t.Fatal("expected error")
	}
}

