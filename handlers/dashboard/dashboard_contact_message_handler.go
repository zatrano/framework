package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/packages/renderer"
	"github.com/zatrano/framework/requests"
	"github.com/zatrano/framework/services"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type DashboardContactMessageHandler struct {
	contactService services.IContactService
}

func NewDashboardContactMessageHandler(contactService services.IContactService) *DashboardContactMessageHandler {
	return &DashboardContactMessageHandler{
		contactService: contactService,
	}
}

func (h *DashboardContactMessageHandler) List(c fiber.Ctx) error {
	params, fieldErrors, err := requests.ParseAndValidateContactMessageList(c)

	paramsMap := fiber.Map{
		"Name":    params.Name,
		"Email":   params.Email,
		"Subject": params.Subject,
		"Unread":  params.Unread,
		"SortBy":  params.SortBy,
		"OrderBy": params.OrderBy,
		"Page":    params.Page,
		"PerPage": params.PerPage,
	}

	if err != nil {
		renderData := fiber.Map{
			"Title":            "İletişim mesajları",
			"ValidationErrors": fieldErrors,
			"Params":           paramsMap,
			"Result":           requests.CreatePaginatedResult([]models.ContactMessage{}, 0, params.Page, params.PerPage),
		}
		return renderer.Render(c, "dashboard/contact-messages/list", "layouts/app", renderData, http.StatusBadRequest)
	}

	result, err := h.contactService.ListMessages(c.Context(), params)
	if err != nil {
		result = requests.CreatePaginatedResult([]models.ContactMessage{}, 0, params.Page, params.PerPage)
	}

	renderData := fiber.Map{
		"Title":  "İletişim mesajları",
		"Result": result,
		"Params": paramsMap,
	}
	if err != nil {
		renderData[renderer.FlashErrorKeyView] = err.Error()
	}
	return renderer.Render(c, "dashboard/contact-messages/list", "layouts/app", renderData, http.StatusOK)
}

func (h *DashboardContactMessageHandler) Show(c fiber.Ctx) error {
	id64, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || id64 == 0 {
		return c.Redirect().To("/dashboard/contact-messages")
	}
	id := uint(id64)
	msg, err := h.contactService.GetMessage(c.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Redirect().To("/dashboard/contact-messages")
		}
		return renderer.Render(c, "dashboard/contact-messages/show", "layouts/app", fiber.Map{
			"Title": "İletişim mesajı",
			"Error": err.Error(),
		}, http.StatusOK)
	}
	_ = h.contactService.MarkMessageRead(c.Context(), id)
	msg, _ = h.contactService.GetMessage(c.Context(), id)

	return renderer.Render(c, "dashboard/contact-messages/show", "layouts/app", fiber.Map{
		"Title":   "İletişim mesajı",
		"Message": msg,
	}, http.StatusOK)
}
