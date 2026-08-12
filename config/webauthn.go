package config

import "github.com/zatrano/framework/packages/env"

// WebAuthn returns WebAuthn/passkey configuration defaults.
func WebAuthn() map[string]any {
	return map[string]any{
		"rp_id":           env.Get("WEBAUTHN_RP_ID", ""),
		"rp_origin":       env.Get("WEBAUTHN_RP_ORIGIN", ""),
		"rp_display_name": env.Get("WEBAUTHN_RP_DISPLAY_NAME", env.Get("WEBAUTHN_RP_NAME", env.Get("APP_NAME", "ZATRANO"))),
	}
}
