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

type DashboardCityHandler struct {
	cityService    services.ICityService
	countryService services.ICountryService
}

func NewDashboardCityHandler(cityService services.ICityService, countryService services.ICountryService) *DashboardCityHandler {
	return &DashboardCityHandler{
		cityService:    cityService,
		countryService: countryService,
	}
}

func (h *DashboardCityHandler) ListCities(c fiber.Ctx) error {
	params, fieldErrors, err := requests.ParseAndValidateCityList(c)
	if err != nil {
		renderData := fiber.Map{
			"Title":            "Şehirler",
			"ValidationErrors": fieldErrors,
			"Params":           params,
			"Result": &queryparams.PaginatedResult{
				Data: []models.City{},
				Meta: queryparams.PaginationMeta{CurrentPage: params.Page, PerPage: params.PerPage, TotalItems: 0, TotalPages: 0},
			},
			"Countries": []models.Country{},
		}
		return renderer.Render(c, "dashboard/cities/list", "layouts/app", renderData, http.StatusBadRequest)
	}

	paginatedResult, err := h.cityService.GetAllCities(c.Context(), params)
	countriesResult, _ := h.countryService.GetAllCountries(c.Context(), requests.CountryListParams{Page: 1, PerPage: 1000, SortBy: "name", OrderBy: "asc"})
	countries := []models.Country{}
	if countriesResult != nil {
		if data, ok := countriesResult.Data.([]models.Country); ok {
			countries = data
		}
	}

	renderData := fiber.Map{
		"Title":     "Şehirler",
		"Result":    paginatedResult,
		"Params":    params,
		"Countries": countries,
	}
	if err != nil {
		renderData[renderer.FlashErrorKeyView] = "Şehirler getirilirken bir hata oluştu."
		renderData["Result"] = &queryparams.PaginatedResult{
			Data: []models.City{},
			Meta: queryparams.PaginationMeta{CurrentPage: params.Page, PerPage: params.PerPage, TotalItems: 0, TotalPages: 0},
		}
	}
	return renderer.Render(c, "dashboard/cities/list", "layouts/app", renderData, http.StatusOK)
}

func (h *DashboardCityHandler) ShowCreateCity(c fiber.Ctx) error {
	countriesResult, _ := h.countryService.GetAllCountries(c.Context(), requests.CountryListParams{Page: 1, PerPage: 1000, SortBy: "name", OrderBy: "asc"})
	countries := []models.Country{}
	if countriesResult != nil {
		if data, ok := countriesResult.Data.([]models.Country); ok {
			countries = data
		}
	}
	return renderer.Render(c, "dashboard/cities/create", "layouts/app", fiber.Map{
		"Title":     "Yeni Şehir Ekle",
		"Countries": countries,
	})
}

func (h *DashboardCityHandler) CreateCity(c fiber.Ctx) error {
	formData := h.getFormData(c)
	req, fieldErrors, err := requests.ParseAndValidateCreateCityRequest(c)
	if err != nil {
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/dashboard/cities/create")
	}
	if err := h.cityService.CreateCity(c.Context(), req); err != nil {
		formflash.SetData(c, formData)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Şehir oluşturulamadı: "+err.Error())
		return c.Redirect().To("/dashboard/cities/create")
	}
	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Şehir başarıyla oluşturuldu.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/cities")
}

func (h *DashboardCityHandler) ShowUpdateCity(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString("Geçersiz şehir ID")
	}
	city, err := h.cityService.GetCityByID(c.Context(), uint(id))
	if err != nil {
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Şehir bulunamadı.")
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/cities")
	}
	countriesResult, _ := h.countryService.GetAllCountries(c.Context(), requests.CountryListParams{Page: 1, PerPage: 1000, SortBy: "name", OrderBy: "asc"})
	countries := []models.Country{}
	if countriesResult != nil {
		if data, ok := countriesResult.Data.([]models.Country); ok {
			countries = data
		}
	}
	return renderer.Render(c, "dashboard/cities/update", "layouts/app", fiber.Map{
		"Title":     "Şehir Düzenle",
		"City":      city,
		"Countries": countries,
	})
}

func (h *DashboardCityHandler) UpdateCity(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz şehir ID")
	}
	formData := h.getFormData(c)
	req, fieldErrors, err := requests.ParseAndValidateUpdateCityRequest(c)
	if err != nil {
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/dashboard/cities/update/" + c.Params("id"))
	}
	if err := h.cityService.UpdateCity(c.Context(), uint(id), req); err != nil {
		formflash.SetData(c, formData)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Şehir güncellenemedi: "+err.Error())
		return c.Redirect().To("/dashboard/cities/update/" + c.Params("id"))
	}
	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Şehir başarıyla güncellendi.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/cities")
}

func (h *DashboardCityHandler) DeleteCity(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz şehir ID")
	}
	if err := h.cityService.DeleteCity(c.Context(), uint(id)); err != nil {
		errMsg := "Şehir silinemedi: " + err.Error()
		if strings.Contains(c.Get("Accept"), "application/json") {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": errMsg})
		}
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, errMsg)
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/cities")
	}
	if strings.Contains(c.Get("Accept"), "application/json") {
		return c.JSON(fiber.Map{"message": "Şehir başarıyla silindi."})
	}
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Şehir başarıyla silindi.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/cities")
}

func (h *DashboardCityHandler) getFormData(c fiber.Ctx) map[string]string {
	formData := make(map[string]string)
	c.Request().PostArgs().VisitAll(func(key, value []byte) { formData[string(key)] = string(value) })
	if form, err := c.MultipartForm(); err == nil {
		for key, values := range form.Value {
			if len(values) > 0 {
				formData[key] = values[0]
			}
		}
	}
	return formData
}
