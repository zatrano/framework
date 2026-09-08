package reference

import (
	"fmt"
	"strings"
	"time"

	"github.com/zatrano/framework/v2/examples/reference/internal/domain"
	"github.com/zatrano/framework/v2/kernel/env"
)

const (
	envName         = "REFERENCE_NAME"
	envPollMS       = "REFERENCE_POLL_MS"
	envAPIToken     = "REFERENCE_API_TOKEN"
	envRequireToken = "REFERENCE_REQUIRE_TOKEN"

	defaultName   = "reference"
	defaultPollMS = 50
)

// Settings is loaded from process environment via kernel/env. There is no
// application ConfigManager.
type Settings struct {
	Name         string
	PollInterval time.Duration
	APIToken     string
}

// LoadSettings reads environment primitives. Invalid integers fail with a
// named type error. Sensitive values are never echoed.
func LoadSettings() (Settings, error) {
	name := env.Get(envName, defaultName)
	ms, err := env.IntOr(envPollMS, defaultPollMS)
	if err != nil {
		return Settings{}, fmt.Errorf("%w: %w", domain.ErrConfiguration, err)
	}
	if ms < 1 {
		return Settings{}, fmt.Errorf("%w: %w", domain.ErrConfiguration, env.ConfigError(envPollMS, "positive integer", fmt.Sprintf("%d", ms)))
	}
	token := env.Get(envAPIToken)
	if env.GetBool(envRequireToken, false) && strings.TrimSpace(token) == "" {
		return Settings{}, fmt.Errorf("%w: configuration error: %s: required", domain.ErrConfiguration, envAPIToken)
	}
	return Settings{
		Name:         name,
		PollInterval: time.Duration(ms) * time.Millisecond,
		APIToken:     token,
	}, nil
}
