package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/packages/flashmessages"
	"github.com/zatrano/framework/packages/formflash"
	"github.com/zatrano/framework/packages/renderer"
	"github.com/zatrano/framework/requests"
	"github.com/zatrano/framework/services"

	"github.com/gofiber/fiber/v3"
)

type DashboardDefinitionHandler struct {
	definitionService services.IDefinitionService
}

func NewDashboardDefinitionHandler(definitionService services.IDefinitionService) *DashboardDefinitionHandler {
	return &DashboardDefinitionHandler{definitionService: definitionService}
}

func (h *DashboardDefinitionHandler) ListDefinitions(c fiber.Ctx) error {
	params, fieldErrors, err := requests.ParseAndValidateDefinitionList(c)
	if err != nil {
		renderData := fiber.Map{
			"Title":            "Tanımlar",
			"ValidationErrors": fieldErrors,
			"Params":           params,
			"Result":           requests.CreatePaginatedResult([]models.Definition{}, 0, params.Page, params.PerPage),
		}
		return renderer.Render(c, "dashboard/definitions/list", "layouts/app", renderData, http.StatusBadRequest)
	}

	result, svcErr := h.definitionService.GetAllDefinitions(c.Context(), params)
	renderData := fiber.Map{
		"Title":  "Tanımlar",
		"Result": result,
		"Params": params,
	}
	if svcErr != nil {
		renderData[renderer.FlashErrorKeyView] = svcErr.Error()
		renderData["Result"] = requests.CreatePaginatedResult([]models.Definition{}, 0, params.Page, params.PerPage)
	}

	return renderer.Render(c, "dashboard/definitions/list", "layouts/app", renderData, http.StatusOK)
}

func (h *DashboardDefinitionHandler) ShowCreateDefinition(c fiber.Ctx) error {
	return renderer.Render(c, "dashboard/definitions/create", "layouts/app", fiber.Map{
		"Title": "Yeni tanım",
	})
}

func (h *DashboardDefinitionHandler) CreateDefinition(c fiber.Ctx) error {
	formData := h.getFormData(c)
	req, fieldErrors, err := requests.ParseAndValidateCreateDefinitionRequest(c)
	if err != nil {
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/dashboard/definitions/create")
	}
	if err := h.definitionService.CreateDefinition(c.Context(), req); err != nil {
		formflash.SetData(c, formData)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/dashboard/definitions/create")
	}
	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Tanım başarıyla oluşturuldu.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/definitions")
}

func (h *DashboardDefinitionHandler) ShowUpdateDefinition(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString("Geçersiz tanım ID")
	}

	def, err := h.definitionService.GetDefinitionByID(c.Context(), uint(id))
	if err != nil {
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/definitions")
	}

	return renderer.Render(c, "dashboard/definitions/update", "layouts/app", fiber.Map{
		"Title":      "Tanım düzenle",
		"Definition": def,
	})
}

func (h *DashboardDefinitionHandler) UpdateDefinition(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz tanım ID")
	}

	formData := h.getFormData(c)
	req, fieldErrors, err := requests.ParseAndValidateUpdateDefinitionRequest(c)
	if err != nil {
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/dashboard/definitions/update/" + c.Params("id"))
	}
	if err := h.definitionService.UpdateDefinition(c.Context(), uint(id), req); err != nil {
		formflash.SetData(c, formData)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/dashboard/definitions/update/" + c.Params("id"))
	}
	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Tanım başarıyla güncellendi.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/definitions")
}

func (h *DashboardDefinitionHandler) DeleteDefinition(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz tanım ID")
	}
	if err := h.definitionService.DeleteDefinition(c.Context(), uint(id)); err != nil {
		errMsg := err.Error()
		if strings.Contains(c.Get("Accept"), "application/json") {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": errMsg})
		}
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, errMsg)
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/definitions")
	}
	if strings.Contains(c.Get("Accept"), "application/json") {
		return c.JSON(fiber.Map{"message": "Tanım başarıyla silindi."})
	}
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Tanım başarıyla silindi.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/definitions")
}

func (h *DashboardDefinitionHandler) getFormData(c fiber.Ctx) map[string]string {
	formData := make(map[string]string)
	c.Request().PostArgs().VisitAll(func(key, value []byte) {
		formData[string(key)] = string(value)
	})
	if form, err := c.MultipartForm(); err == nil {
		for key, values := range form.Value {
			if len(values) > 0 {
				formData[key] = values[0]
			}
		}
	}
	return formData
}
