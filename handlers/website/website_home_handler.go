package handlers

import (
	"github.com/zatrano/framework/packages/renderer"

	"github.com/gofiber/fiber/v3"
)

// WebsiteHomeHandler, genel (public) sitenin ana sayfa işleyicisidir.
// DefinitionValues, global SharedDataMiddleware üzerinden gelir.
type WebsiteHomeHandler struct{}

func NewWebsiteHomeHandler() *WebsiteHomeHandler {
	return &WebsiteHomeHandler{}
}

// Home, ana sayfa.
func (h *WebsiteHomeHandler) Home(c fiber.Ctx) error {
	return renderer.Render(c, "website/home", "layouts/website", fiber.Map{
		"Title": "Ana Sayfa",
	})
}
