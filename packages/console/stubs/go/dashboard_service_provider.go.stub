package providers

import (
	"github.com/zatrano/framework/app/models"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/authorization"
	"github.com/zatrano/framework/packages/view"
)

// DashboardServiceProvider registers dashboard gates and view composers.
type DashboardServiceProvider struct{}

func (p *DashboardServiceProvider) Register(app *core.Application) error {
	return nil
}

func (p *DashboardServiceProvider) Boot(app *core.Application) error {
	if gate := authorization.From(app); gate != nil {
		gate.Define("dashboard.access", func(user authorization.Authenticatable, _ ...any) bool {
			u, ok := user.(*models.User)
			return ok && u != nil && u.IsAdmin
		})
	}

	if v := view.From(app); v != nil {
		v.Composer("dashboard.*", func(_ string, data map[string]any) {
			data["app_shell"] = "dashboard"
		})
		v.Composer("layouts.dashboard", func(_ string, data map[string]any) {
			if _, ok := data["locale"]; !ok {
				data["locale"] = "en"
			}
		})
	}
	return nil
}
