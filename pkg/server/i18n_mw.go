package server

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/framework/pkg/core"
	"github.com/zatrano/framework/pkg/i18n"
)

func registerI18nMiddleware(a *core.App, app *fiber.App) {
	if a.I18n == nil || !a.Config.I18n.Enabled {
		return
	}
	app.Use(i18n.Middleware(a.I18n, a.Config.I18n))
}
