package ai

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ProviderOptions configures a named provider at boot.
type ProviderOptions struct {
	Driver  string
	APIKey  string
	BaseURL string
	Model   string
	Timeout time.Duration
}

// BuildDriver constructs a Driver from provider options.
func BuildDriver(opts ProviderOptions) (Driver, error) {
	driver := strings.ToLower(strings.TrimSpace(opts.Driver))
	if driver == "" {
		driver = "openai"
	}
	switch driver {
	case "fake":
		return FakeDriver{}, nil
	case "log":
		return LogDriver{Inner: FakeDriver{}}, nil
	case "openai":
		d := &OpenAIDriver{
			BaseURL: opts.BaseURL,
			APIKey:  opts.APIKey,
			Model:   opts.Model,
			name:    "openai",
		}
		if d.BaseURL == "" {
			d.BaseURL = "https://api.openai.com/v1"
		}
		if d.Model == "" {
			d.Model = "gpt-4o-mini"
		}
		if opts.Timeout > 0 {
			d.HTTPClient = &http.Client{Timeout: opts.Timeout}
		}
		return d, nil
	case "openai_compatible", "compatible", "ollama", "openrouter":
		d := &OpenAIDriver{
			BaseURL: opts.BaseURL,
			APIKey:  opts.APIKey,
			Model:   opts.Model,
			name:    "openai_compatible",
		}
		if d.BaseURL == "" {
			return nil, fmt.Errorf("ai: openai_compatible requires base_url")
		}
		if d.Model == "" {
			d.Model = "gpt-4o-mini"
		}
		if opts.Timeout > 0 {
			d.HTTPClient = &http.Client{Timeout: opts.Timeout}
		}
		return d, nil
	default:
		return nil, fmt.Errorf("ai: unknown driver %q", opts.Driver)
	}
}

// ParseTemperature parses a config temperature string.
func ParseTemperature(raw string) *float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	t, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	return &t
}

// MapStringAny returns nested map[string]any from config values.
func MapStringAny(v any) map[string]any {
	switch m := v.(type) {
	case map[string]any:
		return m
	default:
		return nil
	}
}

// StringSlice extracts a string slice from config ( []any or []string ).
func StringSlice(v any) []string {
	switch s := v.(type) {
	case []string:
		return append([]string(nil), s...)
	case []any:
		out := make([]string, 0, len(s))
		for _, item := range s {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return nil
	}
}
