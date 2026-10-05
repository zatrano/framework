package kernel

import (
	"strconv"
	"strings"
	"time"

	"github.com/zatrano/framework/v3/core/bootstrap/addons"
	"github.com/zatrano/framework/v3/core/kernel/env"
	"github.com/zatrano/framework/v3/core/kernel/http"
)

// serverTimeout reads a listen timeout from the environment.
//
// Unset or blank keeps fallback (60s read, 60s write, 120s idle, 10s header).
// "0", "0s", and any negative value disable the deadline. rawhttp treats a
// negative duration as unlimited, so those inputs are returned as -1.
// A unitless integer is seconds. Go durations ("30s", "1m") are accepted.
// Anything else is a boot error.
func serverTimeout(key string, fallback time.Duration) (time.Duration, error) {
	raw, ok := env.Lookup(key)
	if !ok {
		return fallback, nil
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	if d, err := time.ParseDuration(raw); err == nil {
		if d <= 0 {
			return -1, nil
		}
		return d, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, env.ConfigError(key, "seconds, a Go duration (30s, 1m), or 0/negative to disable", raw)
	}
	if n <= 0 {
		return -1, nil
	}
	return time.Duration(n) * time.Second, nil
}

// resolveAllowUpgrade decides Server.AllowUpgrade.
//
//  1. HTTP_ALLOW_UPGRADE=false forces admission off.
//  2. A non-nil ListenOptions.AllowUpgrade (including serve --allow-upgrade).
//  3. HTTP_ALLOW_UPGRADE=true.
//  4. RegisterUpgradeProtocol.
//  5. A linked websocket addon.
//  6. EnabledAddons contains "websocket".
//  7. Otherwise off.
func resolveAllowUpgrade(app *Application, opts ListenOptions) bool {
	if _, ok := env.Lookup("HTTP_ALLOW_UPGRADE"); ok && !env.GetBool("HTTP_ALLOW_UPGRADE", false) {
		return false
	}
	if opts.AllowUpgrade != nil {
		return *opts.AllowUpgrade
	}
	if _, ok := env.Lookup("HTTP_ALLOW_UPGRADE"); ok {
		return true
	}
	if http.UpgradeProtocolRegistered() {
		return true
	}
	if _, ok := addons.Lookup("websocket"); ok {
		return true
	}
	if app != nil {
		for _, name := range app.EnabledAddons() {
			if strings.EqualFold(strings.TrimSpace(name), "websocket") {
				return true
			}
		}
	}
	return false
}
