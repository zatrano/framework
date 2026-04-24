package routes

import (
	"github.com/zatrano/framework/app"
	handlers "github.com/zatrano/framework/handlers/website"
	"github.com/gofiber/fiber/v3"
)

func registerWebsiteRoutes(fiberApp *fiber.App, c *app.Container) {
	homeHandler := handlers.NewWebsiteHomeHandler()
	contactHandler := handlers.NewWebsiteContactHandler(c.Contact)
	errHandler := handlers.NewWebsiteErrorHandler()

	websiteGroup := fiberApp.Group("/")
	websiteGroup.Get("/", homeHandler.Home)
	websiteGroup.Get("/iletisim", contactHandler.ContactPage)
	websiteGroup.Post("/iletisim", contactHandler.ContactSubmit)

	fiberApp.Use(errHandler.NotFound)
}
