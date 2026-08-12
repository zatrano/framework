package config

import "github.com/zatrano/framework/packages/env"

// AI returns AI provider configuration defaults.
func AI() map[string]any {
	return map[string]any{
		"driver":  env.Get("AI_DRIVER", ""),
		"api_key": env.Get("AI_API_KEY", env.Get("OPENAI_API_KEY", "")),
	}
}
