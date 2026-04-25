package routes

import (
	"github.com/zatrano/framework/app"
	"github.com/zatrano/framework/middlewares"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiberApp *fiber.App, c *app.Container) {
	fiberApp.Use(middlewares.GlobalRateLimit())
	fiberApp.Use(middlewares.FormPostRateLimit())
	fiberApp.Use(middlewares.SharedDataMiddleware(c.Definition))

	registerAuthRoutes(fiberApp, c)
	registerDashboardRoutes(fiberApp, c)
	registerPanelRoutes(fiberApp, c)
	registerWebsiteRoutes(fiberApp, c)
}
