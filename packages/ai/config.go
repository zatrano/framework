package ai

import "github.com/zatrano/framework/packages/env"

// DefaultConfig returns AI manager configuration defaults.
func DefaultConfig() map[string]any {
	return map[string]any{
		"default":             env.Get("AI_DEFAULT", env.Get("AI_DRIVER", "")),
		"driver":              env.Get("AI_DRIVER", ""),
		"api_key":             env.Get("AI_API_KEY", env.Get("OPENAI_API_KEY", "")),
		"base_url":            env.Get("AI_BASE_URL", ""),
		"model":               env.Get("AI_MODEL", ""),
		"timeout":             env.GetInt("AI_TIMEOUT", 30),
		"temperature":         env.Get("AI_TEMPERATURE", ""),
		"max_tokens":          env.GetInt("AI_MAX_TOKENS", 0),
		"retry_max":           env.GetInt("AI_RETRY_MAX", 2),
		"retry_initial_ms":    env.GetInt("AI_RETRY_INITIAL_MS", 200),
		"retry_max_ms":        env.GetInt("AI_RETRY_MAX_MS", 2000),
		"fallback_on_timeout": env.Get("AI_FALLBACK_ON_TIMEOUT", "true"),
		"providers":           map[string]any{},
		"profiles":            map[string]any{},
	}
}
