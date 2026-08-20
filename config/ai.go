package config

import "github.com/zatrano/framework/packages/env"

// AI returns AI provider configuration defaults.
func AI() map[string]any {
	return map[string]any{
		"driver":      env.Get("AI_DRIVER", ""),
		"api_key":     env.Get("AI_API_KEY", env.Get("OPENAI_API_KEY", "")),
		"base_url":    env.Get("AI_BASE_URL", ""),
		"model":       env.Get("AI_MODEL", ""),
		"timeout":     env.GetInt("AI_TIMEOUT", 30),
		"temperature": env.Get("AI_TEMPERATURE", ""),
		"max_tokens":  env.GetInt("AI_MAX_TOKENS", 0),
	}
}
