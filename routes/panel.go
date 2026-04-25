package routes

import (
	"github.com/zatrano/framework/app"
	handlers "github.com/zatrano/framework/handlers/panel"
	"github.com/zatrano/framework/middlewares"

	"github.com/gofiber/fiber/v3"
)

func registerPanelRoutes(fiberApp *fiber.App, c *app.Container) {
	panelGroup := fiberApp.Group("/panel")
	panelGroup.Use(
		middlewares.SessionMiddleware(),
		middlewares.AuthMiddleware(c.Auth),
	)

	panelHandler := handlers.NewPanelHomeHandler()
	panelGroup.Get("/", func(ctx fiber.Ctx) error {
		return ctx.Redirect().Status(fiber.StatusFound).To("/panel/anasayfa")
	})
	panelGroup.Get("/anasayfa", panelHandler.HomePage)
}
