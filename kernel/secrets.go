package kernel

import (
	"fmt"
	"strings"

	"github.com/zatrano/framework/kernel/encryption"
	"github.com/zatrano/framework/kernel/env"
)

// ensureProductionSecrets refuses insecure defaults when APP_ENV=production.
func ensureProductionSecrets(app *Application) error {
	if !app.IsProduction() {
		return nil
	}
	key := strings.TrimSpace(app.config.GetString("app.key", env.Get("APP_KEY", "")))
	switch key {
	case "", encryption.LocalDevKey, "zatrano-dev-key", "changeme", "secret", "password":
		return fmt.Errorf("refusing to boot: set a strong APP_KEY in production")
	}
	if _, err := encryption.New(key); err != nil {
		return fmt.Errorf("refusing to boot: %w", err)
	}
	return nil
}

func (app *Application) appKey() string {
	if app == nil {
		return encryption.LocalDevKey
	}
	key := strings.TrimSpace(app.config.GetString("app.key"))
	if key == "" {
		key = strings.TrimSpace(env.Get("APP_KEY", ""))
	}
	if key == "" {
		return encryption.LocalDevKey
	}
	return key
}
