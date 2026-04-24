package handlers

import (
	"strconv"
	"net/http"
	"strings"

	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/packages/flashmessages"
	"github.com/zatrano/framework/packages/formflash"
	"github.com/zatrano/framework/packages/queryparams"
	"github.com/zatrano/framework/packages/renderer"
	"github.com/zatrano/framework/requests"
	"github.com/zatrano/framework/services"

	"github.com/gofiber/fiber/v3"
)

type DashboardCountryHandler struct {
	countryService services.ICountryService
}

func NewDashboardCountryHandler(countryService services.ICountryService) *DashboardCountryHandler {
	return &DashboardCountryHandler{countryService: countryService}
}

func (h *DashboardCountryHandler) ListCountries(c fiber.Ctx) error {
	params, fieldErrors, err := requests.ParseAndValidateCountryList(c)
	if err != nil {
		renderData := fiber.Map{
			"Title":            "Ülkeler",
			"ValidationErrors": fieldErrors,
			"Params":           params,
			"Result": &queryparams.PaginatedResult{
				Data: []models.Country{},
				Meta: queryparams.PaginationMeta{
					CurrentPage: params.Page,
					PerPage:     params.PerPage,
					TotalItems:  0,
					TotalPages:  0,
				},
			},
		}
		return renderer.Render(c, "dashboard/countries/list", "layouts/app", renderData, http.StatusBadRequest)
	}

	paginatedResult, err := h.countryService.GetAllCountries(c.Context(), params)
	renderData := fiber.Map{
		"Title":  "Ülkeler",
		"Result": paginatedResult,
		"Params": params,
	}
	if err != nil {
		renderData[renderer.FlashErrorKeyView] = "Ülkeler getirilirken bir hata oluştu."
		renderData["Result"] = &queryparams.PaginatedResult{
			Data: []models.Country{},
			Meta: queryparams.PaginationMeta{
				CurrentPage: params.Page,
				PerPage:     params.PerPage,
				TotalItems:  0,
				TotalPages:  0,
			},
		}
	}
	return renderer.Render(c, "dashboard/countries/list", "layouts/app", renderData, http.StatusOK)
}

func (h *DashboardCountryHandler) ShowCreateCountry(c fiber.Ctx) error {
	return renderer.Render(c, "dashboard/countries/create", "layouts/app", fiber.Map{
		"Title": "Yeni Ülke Ekle",
	})
}

func (h *DashboardCountryHandler) CreateCountry(c fiber.Ctx) error {
	formData := h.getFormData(c)
	req, fieldErrors, err := requests.ParseAndValidateCreateCountryRequest(c)
	if err != nil {
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/dashboard/countries/create")
	}
	if err := h.countryService.CreateCountry(c.Context(), req); err != nil {
		formflash.SetData(c, formData)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Ülke oluşturulamadı: "+err.Error())
		return c.Redirect().To("/dashboard/countries/create")
	}
	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Ülke başarıyla oluşturuldu.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/countries")
}

func (h *DashboardCountryHandler) ShowUpdateCountry(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString("Geçersiz ülke ID")
	}
	country, err := h.countryService.GetCountryByID(c.Context(), uint(id))
	if err != nil {
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Ülke bulunamadı.")
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/countries")
	}
	return renderer.Render(c, "dashboard/countries/update", "layouts/app", fiber.Map{
		"Title":   "Ülke Düzenle",
		"Country": country,
	})
}

func (h *DashboardCountryHandler) UpdateCountry(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz ülke ID")
	}
	formData := h.getFormData(c)
	req, fieldErrors, err := requests.ParseAndValidateUpdateCountryRequest(c)
	if err != nil {
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/dashboard/countries/update/" + c.Params("id"))
	}
	if err := h.countryService.UpdateCountry(c.Context(), uint(id), req); err != nil {
		formflash.SetData(c, formData)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Ülke güncellenemedi: "+err.Error())
		return c.Redirect().To("/dashboard/countries/update/" + c.Params("id"))
	}
	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Ülke başarıyla güncellendi.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/countries")
}

func (h *DashboardCountryHandler) DeleteCountry(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz ülke ID")
	}
	if err := h.countryService.DeleteCountry(c.Context(), uint(id)); err != nil {
		errMsg := "Ülke silinemedi: " + err.Error()
		if strings.Contains(c.Get("Accept"), "application/json") {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": errMsg})
		}
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, errMsg)
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/countries")
	}
	if strings.Contains(c.Get("Accept"), "application/json") {
		return c.JSON(fiber.Map{"message": "Ülke başarıyla silindi."})
	}
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Ülke başarıyla silindi.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/countries")
}

func (h *DashboardCountryHandler) getFormData(c fiber.Ctx) map[string]string {
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
