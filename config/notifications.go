package config

import "github.com/zatrano/framework/packages/env"

// Notifications returns notification configuration.
func Notifications() map[string]any {
	return map[string]any{
		"default_channels": env.Get("NOTIFICATION_CHANNELS", "database,mail"),
		"sms_from":         env.Get("SMS_FROM", env.Get("APP_NAME", "ZATRANO")),
		"sms_driver":       env.Get("SMS_DRIVER", "memory"),
	}
}
