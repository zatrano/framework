package config

import "github.com/zatrano/framework/packages/env"

// OAuth returns OAuth2 server configuration defaults.
func OAuth() map[string]any {
	return map[string]any{
		"store_path": env.Get("OAUTH_STORE_PATH", ""),
	}
}
