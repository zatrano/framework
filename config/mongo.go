package config

import "github.com/zatrano/framework/packages/env"

// Mongo returns MongoDB configuration defaults.
func Mongo() map[string]any {
	return map[string]any{
		"uri": env.Get("MONGO_URI", "memory"),
	}
}
