package config

import "github.com/zatrano/framework/packages/env"

// Billing returns billing configuration defaults.
func Billing() map[string]any {
	return map[string]any{
		"stripe_secret": env.Get("STRIPE_SECRET_KEY", ""),
	}
}
