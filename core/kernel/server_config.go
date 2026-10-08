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

// DefaultMaxHeaderBytes is the production header-block ceiling.
// rawhttp sizes the per-connection read buffer from this value, so the
// v3.0.0–v3.0.1 default of 1 MiB made connection setup allocate and zero a
// megabyte on every new connection.
const DefaultMaxHeaderBytes = 16 << 10

// MaxHeaderBytesWarnAbove is the size that logs a startup warning.
// nginx and Apache keep a single header line near 8 KiB. Above 64 KiB the
// connection buffer cost is worth saying out loud until rawhttp grows the
// buffer from 4 KiB.
const MaxHeaderBytesWarnAbove = 64 << 10

// serverMaxHeaderBytes reads HTTP_MAX_HEADER_BYTES.
// Unset or blank uses DefaultMaxHeaderBytes. The value is a unitless byte
// count. Zero, negative, and non-integers are boot errors.
func serverMaxHeaderBytes() (int, error) {
	n, err := env.IntOr("HTTP_MAX_HEADER_BYTES", DefaultMaxHeaderBytes)
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		raw, _ := env.Lookup("HTTP_MAX_HEADER_BYTES")
		return 0, env.ConfigError("HTTP_MAX_HEADER_BYTES", "a positive number of bytes", strings.TrimSpace(raw))
	}
	return n, nil
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
