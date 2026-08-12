package config

import "github.com/zatrano/framework/packages/env"

// Social returns social login configuration defaults.
func Social() map[string]any {
	return map[string]any{
		"github_client_id":     env.Get("GITHUB_CLIENT_ID", ""),
		"github_client_secret": env.Get("GITHUB_CLIENT_SECRET", ""),
		"github_redirect_uri":  env.Get("GITHUB_REDIRECT_URI", ""),
		"google_client_id":     env.Get("GOOGLE_CLIENT_ID", ""),
		"google_client_secret": env.Get("GOOGLE_CLIENT_SECRET", ""),
		"google_redirect_uri":  env.Get("GOOGLE_REDIRECT_URI", ""),
	}
}
