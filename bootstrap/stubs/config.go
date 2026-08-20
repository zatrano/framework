package stubs

// Files maps config filename -> stub contents for package:publish.
var Files = map[string]string{
	"oauth.go": `package config

import "github.com/zatrano/framework/packages/env"

// OAuth returns OAuth2 server configuration defaults.
func OAuth() map[string]any {
	return map[string]any{
		"store_path": env.Get("OAUTH_STORE_PATH", ""),
	}
}
`,
	"mongo.go": `package config

import "github.com/zatrano/framework/packages/env"

// Mongo returns MongoDB configuration defaults.
func Mongo() map[string]any {
	return map[string]any{
		"uri": env.Get("MONGO_URI", "memory"),
	}
}
`,
	"webauthn.go": `package config

import "github.com/zatrano/framework/packages/env"

// WebAuthn returns WebAuthn/passkey configuration defaults.
func WebAuthn() map[string]any {
	return map[string]any{
		"rp_id":           env.Get("WEBAUTHN_RP_ID", ""),
		"rp_origin":       env.Get("WEBAUTHN_RP_ORIGIN", ""),
		"rp_display_name": env.Get("WEBAUTHN_RP_DISPLAY_NAME", env.Get("WEBAUTHN_RP_NAME", env.Get("APP_NAME", "ZATRANO"))),
	}
}
`,
	"billing.go": `package config

import "github.com/zatrano/framework/packages/env"

// Billing returns billing configuration defaults.
func Billing() map[string]any {
	return map[string]any{
		"default":               env.Get("BILLING_GATEWAY", "memory"),
		"stripe_secret":         env.Get("STRIPE_SECRET_KEY", ""),
		"stripe_webhook_secret": env.Get("STRIPE_WEBHOOK_SECRET", ""),
		"success_url":           env.Get("BILLING_SUCCESS_URL", ""),
		"cancel_url":            env.Get("BILLING_CANCEL_URL", ""),
	}
}
`,
	"ai.go": `package config

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
`,
	"social.go": `package config

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
`,
}

// ForPackage returns stub filenames useful for a given addon name.
func ForPackage(name string) []string {
	switch name {
	case "oauth":
		return []string{"oauth.go"}
	case "mongo":
		return []string{"mongo.go"}
	case "webauthn":
		return []string{"webauthn.go"}
	case "billing":
		return []string{"billing.go"}
	case "ai":
		return []string{"ai.go"}
	case "social":
		return []string{"social.go"}
	default:
		return nil
	}
}
