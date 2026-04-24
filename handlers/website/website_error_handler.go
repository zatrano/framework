package handlers

import (
	"github.com/zatrano/framework/packages/renderer"

	"github.com/gofiber/fiber/v3"
)

// WebsiteErrorHandler, public website hata/404 cevapları.
// DefinitionValues, global SharedDataMiddleware + renderer.prepareRenderData üzerinden gelir.
type WebsiteErrorHandler struct{}

func NewWebsiteErrorHandler() *WebsiteErrorHandler {
	return &WebsiteErrorHandler{}
}

// NotFound, tanımsız yollar için 404 (router sonunda kullanılır).
func (h *WebsiteErrorHandler) NotFound(c fiber.Ctx) error {
	return h.RenderError(c, fiber.Map{
		"Code":    404,
		"Title":   "Sayfa Bulunamadı",
		"Message": "Aradığınız sayfa mevcut değil veya taşınmış olabilir.",
	}, fiber.StatusNotFound)
}

// RenderError, website layout ile hata sayfası.
func (h *WebsiteErrorHandler) RenderError(c fiber.Ctx, data fiber.Map, status int) error {
	if data == nil {
		data = fiber.Map{}
	}
	return renderer.Render(c, "website/error", "layouts/website", data, status)
}
