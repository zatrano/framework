package web

import (
	"github.com/zatrano/framework/app/models"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/flash"
	. "github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/orm"
)

const pathDashboardSettings = "/dashboard/settings"

// DashboardSettingsController manages key/value settings.
type DashboardSettingsController struct {
	App *core.Application
}

func (c *DashboardSettingsController) Index(req *Request) *Response {
	settings, err := orm.Query[models.Setting]().OrderBy("key").Get()
	if err != nil {
		return flash.WithError(req, dashboardLang(c.App, "dashboard.load_failed"), "/dashboard")
	}
	if len(settings) == 0 {
		defaults := []models.Setting{
			{Key: "app.name", Value: "ZATRANO"},
			{Key: "app.timezone", Value: "UTC"},
		}
		for i := range defaults {
			_ = orm.Save(&defaults[i])
		}
		settings, _ = orm.Query[models.Setting]().OrderBy("key").Get()
	}
	return View("dashboard/settings/index", map[string]any{
		"title":    dashboardLang(c.App, "dashboard.nav_settings"),
		"settings": settings,
		"user":     dashboardCurrentUser(req, c.App),
	})
}

func (c *DashboardSettingsController) Save(req *Request) *Response {
	settings, _ := orm.Query[models.Setting]().Get()
	if len(settings) == 0 {
		key := req.Input("key")
		if key != "" {
			_ = orm.Save(&models.Setting{Key: key, Value: req.Input("value")})
		}
		return flash.WithSuccess(req, dashboardLang(c.App, "dashboard.saved"), pathDashboardSettings)
	}
	for i := range settings {
		field := "setting__" + settings[i].Key
		if req.Has(field) {
			settings[i].Value = req.Input(field)
			_ = orm.Save(&settings[i])
		}
	}
	return flash.WithSuccess(req, dashboardLang(c.App, "dashboard.saved"), pathDashboardSettings)
}
