package kernel

import (
	"strings"

	"github.com/zatrano/framework/v3/core/bootstrap/addons"
	"github.com/zatrano/framework/v3/core/kernel/env"
	"github.com/zatrano/framework/v3/core/kernel/http"
)

// resolveAllowUpgrade decides Server.AllowUpgrade.
//
//	1. HTTP_ALLOW_UPGRADE=false forces admission off.
//	2. A non-nil ListenOptions.AllowUpgrade (including serve --allow-upgrade).
//	3. HTTP_ALLOW_UPGRADE=true.
//	4. RegisterUpgradeProtocol.
//	5. A linked websocket addon.
//	6. EnabledAddons contains "websocket".
//	7. Otherwise off.
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
