package kernel

import (
	"fmt"
	"strings"

	"github.com/zatrano/framework/kernel/env"
)

// ensureProductionSecrets refuses insecure defaults when APP_ENV=production.
func ensureProductionSecrets(app *Application) error {
	if !app.IsProduction() {
		return nil
	}
	key := strings.TrimSpace(app.config.GetString("app.key", env.Get("APP_KEY", "")))
	switch key {
	case "", "zatrano-dev-key", "changeme", "secret":
		return fmt.Errorf("refusing to boot: set a strong APP_KEY in production")
	}
	return nil
}
