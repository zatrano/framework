package kernel

import (
	"fmt"
	"strings"

	"github.com/zatrano/framework/v3/core/kernel/encryption"
	"github.com/zatrano/framework/v3/core/kernel/env"
)

// Each production refusal below is turned off only by its own key: the key,
// debug and cookie checks by the value "allow", the CSRF check by
// CSRF_DISABLED=true. There is no shared "insecure mode" switch. The empty,
// known-default and wrong-length APP_KEY refusals have no opt-out.
const (
	allowWeakAppKey          = "APP_KEY_WEAK_IN_PRODUCTION"
	allowDebugInProduction   = "APP_DEBUG_IN_PRODUCTION"
	allowInsecureCookies     = "COOKIE_INSECURE_IN_PRODUCTION"
	allowCSRFDisabled        = "CSRF_DISABLED"
	allowValue               = "allow"
	csrfDisabledAcknowledged = "true"
)

// ensureProductionConfig refuses a production boot whose configuration
// turns off a protection without saying so. It runs after providers Boot so
// the routes and middleware they register are visible.
func ensureProductionConfig(app *Application) error {
	if app == nil || !app.IsProduction() {
		return nil
	}
	if app.IsDebug() && env.Get(allowDebugInProduction, "") != allowValue {
		return fmt.Errorf("refusing to boot: APP_DEBUG is on in production; set APP_DEBUG=false (or %s=%s to accept debug pages in production)", allowDebugInProduction, allowValue)
	}
	if err := insecureCookieConfig(); err != nil {
		return err
	}
	if err := csrfOffWithoutConsent(app); err != nil {
		return err
	}
	return nil
}

// weakAppKey reports whether a key that parses as an AES key is still too
// predictable for production: a short repeating pattern or a run of the same
// few characters.
func weakAppKey(raw string) bool {
	key, err := encryption.KeyBytes(raw)
	if err != nil {
		return true
	}
	seen := [256]bool{}
	distinct := 0
	for _, b := range key {
		if !seen[b] {
			seen[b] = true
			distinct++
		}
	}
	return distinct*2 < len(key)
}

func insecureCookieConfig() error {
	appURL := strings.ToLower(strings.TrimSpace(env.Get("APP_URL", "")))
	if !strings.HasPrefix(appURL, "https://") {
		return nil
	}
	if env.Get(allowInsecureCookies, "") == allowValue {
		return nil
	}
	for _, key := range []string{"COOKIE_SECURE", "SESSION_SECURE"} {
		if v, ok := env.Lookup(key); ok && isFalse(v) {
			return fmt.Errorf("refusing to boot: %s=%s sends cookies over plain HTTP while APP_URL is https; remove it (or set %s=%s)", key, strings.TrimSpace(v), allowInsecureCookies, allowValue)
		}
	}
	return nil
}

// csrfOffWithoutConsent refuses a production boot that installs a session
// bridge but no CSRF middleware. A bridge that adds middleware carries
// cookie sessions; CSRF_DISABLED=true records that an API-only or
// token-authenticated application does not need the check.
func csrfOffWithoutConsent(app *Application) error {
	bridge := app.httpBridgeCaptured
	if bridge == nil || len(bridge.Middleware()) == 0 {
		return nil
	}
	if app.csrfComposed {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(env.Get(allowCSRFDisabled, "")), csrfDisabledAcknowledged) {
		return nil
	}
	return fmt.Errorf("refusing to boot: sessions are on but no CSRF middleware is installed; add middleware.CSRF or CSRFExcept (or set %s=%s)", allowCSRFDisabled, csrfDisabledAcknowledged)
}

func isFalse(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "false", "0", "off", "no":
		return true
	}
	return false
}
