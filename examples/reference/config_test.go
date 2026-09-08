package reference

import (
	"errors"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/examples/reference/internal/domain"
	"github.com/zatrano/framework/v2/kernel/env"
)

func TestLoadSettingsDefaults(t *testing.T) {
	t.Setenv(envName, defaultName)
	t.Setenv(envPollMS, "")
	t.Setenv(envAPIToken, "")
	t.Setenv(envRequireToken, "false")
	s, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != defaultName {
		t.Fatalf("name=%q", s.Name)
	}
	if s.PollInterval.Milliseconds() != defaultPollMS {
		t.Fatalf("poll=%s", s.PollInterval)
	}
}

func TestLoadSettingsIntegerParsing(t *testing.T) {
	t.Setenv(envPollMS, "25")
	s, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.PollInterval.Milliseconds() != 25 {
		t.Fatalf("poll=%s", s.PollInterval)
	}
}

func TestLoadSettingsInvalidInteger(t *testing.T) {
	t.Setenv(envPollMS, "not-an-int")
	_, err := LoadSettings()
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, domain.ErrConfiguration) {
		t.Fatalf("want ErrConfiguration, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, envPollMS) || !strings.Contains(msg, "integer") {
		t.Fatalf("msg=%q", msg)
	}
	if !strings.Contains(msg, "not-an-int") {
		t.Fatalf("non-sensitive value should be named: %q", msg)
	}
}

func TestLoadSettingsRequiredTokenDoesNotEchoValue(t *testing.T) {
	t.Setenv(envRequireToken, "true")
	t.Setenv(envAPIToken, "")
	_, err := LoadSettings()
	if err == nil {
		t.Fatal("expected required-token error")
	}
	if !errors.Is(err, domain.ErrConfiguration) {
		t.Fatalf("want ErrConfiguration, got %v", err)
	}
	if !strings.Contains(err.Error(), envAPIToken) {
		t.Fatalf("msg=%q", err.Error())
	}
}

func TestSensitiveInvalidIntegerDoesNotEchoValue(t *testing.T) {
	secret := "super-secret-value-xyz"
	t.Setenv(envAPIToken, secret)
	_, err := env.IntOr(envAPIToken, 0)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("secret leaked: %v", err)
	}
	if !strings.Contains(err.Error(), envAPIToken) {
		t.Fatalf("key missing: %v", err)
	}
}

func TestInvalidPollDoesNotIncludeToken(t *testing.T) {
	secret := "super-secret-value-xyz"
	t.Setenv(envAPIToken, secret)
	t.Setenv(envPollMS, "abc")
	_, err := LoadSettings()
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("token leaked: %v", err)
	}
}
