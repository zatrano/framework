package ai

import (
	"fmt"
	"strings"
	"time"
)

// BootConfig loads providers/profiles from a config map (as returned by config.AI()).
func (m *Manager) BootConfig(cfg map[string]any, log LogFn) error {
	if m == nil {
		return fmt.Errorf("ai: manager is nil")
	}
	if cfg == nil {
		cfg = map[string]any{}
	}

	timeoutSec := asInt(cfg["timeout"], 30)
	timeout := time.Duration(timeoutSec) * time.Second
	retry := DefaultRetryPolicy()
	if v, ok := cfg["retry_max"]; ok {
		retry.MaxRetries = asInt(v, retry.MaxRetries)
	}
	if v, ok := cfg["retry_initial_ms"]; ok {
		ms := asInt(v, int(retry.InitialDelay/time.Millisecond))
		retry.InitialDelay = time.Duration(ms) * time.Millisecond
	}
	if v, ok := cfg["retry_max_ms"]; ok {
		ms := asInt(v, int(retry.MaxDelay/time.Millisecond))
		retry.MaxDelay = time.Duration(ms) * time.Millisecond
	}
	fallbackOnTimeout := true
	if v, ok := cfg["fallback_on_timeout"]; ok {
		fallbackOnTimeout = asBool(v, true)
	}
	defs := Defaults{
		Model:             asString(cfg["model"]),
		MaxTokens:         asInt(cfg["max_tokens"], 0),
		Timeout:           timeout,
		Retry:             retry,
		FallbackOnTimeout: fallbackOnTimeout,
	}
	if t := ParseTemperature(asString(cfg["temperature"])); t != nil {
		defs.Temperature = t
	}
	m.SetDefaults(defs)

	if log != nil {
		m.Extend("log", LogDriver{Log: log, Inner: FakeDriver{}})
	}

	providers := MapStringAny(cfg["providers"])
	registeredNamed := false
	for name, raw := range providers {
		optsMap := MapStringAny(raw)
		if optsMap == nil {
			continue
		}
		opts := ProviderOptions{
			Driver:  asString(optsMap["driver"]),
			APIKey:  asString(optsMap["api_key"]),
			BaseURL: asString(optsMap["base_url"]),
			Model:   firstNonEmpty(asString(optsMap["model"]), defs.Model),
			Timeout: timeout,
		}
		if opts.Driver == "" {
			opts.Driver = "openai"
		}
		drv, err := BuildDriver(opts)
		if err != nil {
			return fmt.Errorf("ai: provider %q: %w", name, err)
		}
		m.Extend(name, drv)
		registeredNamed = true
	}

	// Legacy flat config → single provider when no named providers.
	if !registeredNamed {
		apiKey := asString(cfg["api_key"])
		if apiKey != "" {
			opts := ProviderOptions{
				Driver:  "openai",
				APIKey:  apiKey,
				BaseURL: asString(cfg["base_url"]),
				Model:   firstNonEmpty(defs.Model, "gpt-4o-mini"),
				Timeout: timeout,
			}
			if strings.TrimSpace(asString(cfg["base_url"])) != "" &&
				!strings.Contains(strings.ToLower(asString(cfg["base_url"])), "api.openai.com") {
				opts.Driver = "openai_compatible"
			}
			drv, err := BuildDriver(opts)
			if err != nil {
				return err
			}
			m.Extend("openai", drv)
			m.Use("openai")
		}
	}

	profiles := MapStringAny(cfg["profiles"])
	for name, raw := range profiles {
		pm := MapStringAny(raw)
		if pm == nil {
			continue
		}
		p := Profile{
			Providers: StringSlice(pm["providers"]),
			Model:     asString(pm["model"]),
			MaxTokens: asInt(pm["max_tokens"], 0),
		}
		if len(p.Providers) == 0 {
			if single := asString(pm["provider"]); single != "" {
				p.Providers = []string{single}
			}
		}
		if t := ParseTemperature(asString(pm["temperature"])); t != nil {
			p.Temperature = t
		}
		m.SetProfile(name, p)
	}

	def := firstNonEmpty(asString(cfg["default"]), asString(cfg["driver"]))
	if def != "" {
		m.Use(def)
	}
	return nil
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func asInt(v any, fallback int) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		if n, err := strconvAtoi(n); err == nil {
			return n
		}
	}
	return fallback
}

func asBool(v any, fallback bool) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		s := strings.ToLower(strings.TrimSpace(b))
		switch s {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	case int:
		return b != 0
	case float64:
		return b != 0
	}
	return fallback
}

func strconvAtoi(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n, err
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
