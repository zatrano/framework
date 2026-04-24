package handlers

import (
	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/packages/flashmessages"
	"github.com/zatrano/framework/packages/formflash"
	"github.com/zatrano/framework/packages/renderer"
	"github.com/zatrano/framework/requests"
	"github.com/zatrano/framework/services"

	"github.com/gofiber/fiber/v3"
)

type WebsiteContactHandler struct {
	contactService services.IContactService
}

func NewWebsiteContactHandler(contactService services.IContactService) *WebsiteContactHandler {
	return &WebsiteContactHandler{contactService: contactService}
}

// ContactPage iletişim sayfası (Cloudflare Turnstile site key ile)
func (h *WebsiteContactHandler) ContactPage(c fiber.Ctx) error {
	contactData := fiber.Map{
		"Title":            "İletişim",
		"TurnstileSiteKey": envconfig.String("TURNSTILE_SITE_KEY", ""),
	}
	return renderer.Render(c, "website/contact", "layouts/website", contactData)
}

// ContactSubmit — validasyon, Turnstile (yapılandırılmışsa), veritabanına kayıt
func (h *WebsiteContactHandler) ContactSubmit(c fiber.Ctx) error {
	req := requests.ContactSubmitRequest{
		Name:              c.FormValue("name"),
		Email:             c.FormValue("email"),
		Phone:             c.FormValue("phone"),
		Subject:           c.FormValue("subject"),
		Message:           c.FormValue("message"),
		TurnstileResponse: c.FormValue("cf-turnstile-response"),
	}

	if errs := req.Validate(); len(errs) > 0 {
		_ = formflash.SetData(c, req.ToFormData())
		_ = formflash.SetValidationErrors(c, errs)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Lütfen formu kontrol edip tekrar deneyin.")
		return c.Redirect().Status(fiber.StatusFound).To("/iletisim")
	}

	if err := h.contactService.Submit(c.Context(), req, c.IP(), c.Get("User-Agent")); err != nil {
		_ = formflash.SetData(c, req.ToFormData())
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().Status(fiber.StatusFound).To("/iletisim")
	}

	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Mesajınız alındı. En kısa sürede size dönüş yapacağız.")
	return c.Redirect().Status(fiber.StatusFound).To("/iletisim")
}
