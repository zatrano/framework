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

type DashboardDistrictHandler struct {
	districtService services.IDistrictService
	cityService     services.ICityService
	countryService  services.ICountryService
}

func NewDashboardDistrictHandler(
	districtService services.IDistrictService,
	cityService services.ICityService,
	countryService services.ICountryService,
) *DashboardDistrictHandler {
	return &DashboardDistrictHandler{
		districtService: districtService,
		cityService:     cityService,
		countryService:  countryService,
	}
}

func (h *DashboardDistrictHandler) ListDistricts(c fiber.Ctx) error {
	params, fieldErrors, err := requests.ParseAndValidateDistrictList(c)
	if err != nil {
		renderData := fiber.Map{
			"Title":            "İlçeler",
			"ValidationErrors": fieldErrors,
			"Params":           params,
			"Result": &queryparams.PaginatedResult{
				Data: []models.District{},
				Meta: queryparams.PaginationMeta{CurrentPage: params.Page, PerPage: params.PerPage, TotalItems: 0, TotalPages: 0},
			},
			"Countries": []models.Country{},
			"Cities":    []models.City{},
		}
		return renderer.Render(c, "dashboard/districts/list", "layouts/app", renderData, http.StatusBadRequest)
	}

	paginatedResult, err := h.districtService.GetAllDistricts(c.Context(), params)
	countriesResult, _ := h.countryService.GetAllCountries(c.Context(), requests.CountryListParams{Page: 1, PerPage: 1000, SortBy: "name", OrderBy: "asc"})
	countries := []models.Country{}
	if countriesResult != nil {
		if data, ok := countriesResult.Data.([]models.Country); ok {
			countries = data
		}
	}
	citiesResult, _ := h.cityService.GetAllCities(c.Context(), requests.CityListParams{Page: 1, PerPage: 1000, SortBy: "name", OrderBy: "asc"})
	cities := []models.City{}
	if citiesResult != nil {
		if data, ok := citiesResult.Data.([]models.City); ok {
			cities = data
		}
	}

	renderData := fiber.Map{
		"Title":     "İlçeler",
		"Result":    paginatedResult,
		"Params":    params,
		"Countries": countries,
		"Cities":    cities,
	}
	if err != nil {
		renderData[renderer.FlashErrorKeyView] = "İlçeler getirilirken bir hata oluştu."
		renderData["Result"] = &queryparams.PaginatedResult{
			Data: []models.District{},
			Meta: queryparams.PaginationMeta{CurrentPage: params.Page, PerPage: params.PerPage, TotalItems: 0, TotalPages: 0},
		}
	}
	return renderer.Render(c, "dashboard/districts/list", "layouts/app", renderData, http.StatusOK)
}

func (h *DashboardDistrictHandler) ShowCreateDistrict(c fiber.Ctx) error {
	countriesResult, _ := h.countryService.GetAllCountries(c.Context(), requests.CountryListParams{Page: 1, PerPage: 1000, SortBy: "name", OrderBy: "asc"})
	countries := []models.Country{}
	if countriesResult != nil {
		if data, ok := countriesResult.Data.([]models.Country); ok {
			countries = data
		}
	}
	citiesResult, _ := h.cityService.GetAllCities(c.Context(), requests.CityListParams{Page: 1, PerPage: 1000, SortBy: "name", OrderBy: "asc"})
	cities := []models.City{}
	if citiesResult != nil {
		if data, ok := citiesResult.Data.([]models.City); ok {
			cities = data
		}
	}
	return renderer.Render(c, "dashboard/districts/create", "layouts/app", fiber.Map{
		"Title":     "Yeni İlçe Ekle",
		"Countries": countries,
		"Cities":    cities,
	})
}

func (h *DashboardDistrictHandler) CreateDistrict(c fiber.Ctx) error {
	formData := h.getFormData(c)
	req, fieldErrors, err := requests.ParseAndValidateCreateDistrictRequest(c)
	if err != nil {
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/dashboard/districts/create")
	}
	if err := h.districtService.CreateDistrict(c.Context(), req); err != nil {
		formflash.SetData(c, formData)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "İlçe oluşturulamadı: "+err.Error())
		return c.Redirect().To("/dashboard/districts/create")
	}
	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "İlçe başarıyla oluşturuldu.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/districts")
}

func (h *DashboardDistrictHandler) ShowUpdateDistrict(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString("Geçersiz ilçe ID")
	}
	district, err := h.districtService.GetDistrictByID(c.Context(), uint(id))
	if err != nil {
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "İlçe bulunamadı.")
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/districts")
	}
	countriesResult, _ := h.countryService.GetAllCountries(c.Context(), requests.CountryListParams{Page: 1, PerPage: 1000, SortBy: "name", OrderBy: "asc"})
	countries := []models.Country{}
	if countriesResult != nil {
		if data, ok := countriesResult.Data.([]models.Country); ok {
			countries = data
		}
	}
	citiesResult, _ := h.cityService.GetAllCities(c.Context(), requests.CityListParams{Page: 1, PerPage: 1000, SortBy: "name", OrderBy: "asc"})
	cities := []models.City{}
	if citiesResult != nil {
		if data, ok := citiesResult.Data.([]models.City); ok {
			cities = data
		}
	}
	return renderer.Render(c, "dashboard/districts/update", "layouts/app", fiber.Map{
		"Title":     "İlçe Düzenle",
		"District":  district,
		"Countries": countries,
		"Cities":    cities,
	})
}

func (h *DashboardDistrictHandler) UpdateDistrict(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz ilçe ID")
	}
	formData := h.getFormData(c)
	req, fieldErrors, err := requests.ParseAndValidateUpdateDistrictRequest(c)
	if err != nil {
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/dashboard/districts/update/" + c.Params("id"))
	}
	if err := h.districtService.UpdateDistrict(c.Context(), uint(id), req); err != nil {
		formflash.SetData(c, formData)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "İlçe güncellenemedi: "+err.Error())
		return c.Redirect().To("/dashboard/districts/update/" + c.Params("id"))
	}
	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "İlçe başarıyla güncellendi.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/districts")
}

func (h *DashboardDistrictHandler) DeleteDistrict(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz ilçe ID")
	}
	if err := h.districtService.DeleteDistrict(c.Context(), uint(id)); err != nil {
		errMsg := "İlçe silinemedi: " + err.Error()
		if strings.Contains(c.Get("Accept"), "application/json") {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": errMsg})
		}
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, errMsg)
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/districts")
	}
	if strings.Contains(c.Get("Accept"), "application/json") {
		return c.JSON(fiber.Map{"message": "İlçe başarıyla silindi."})
	}
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "İlçe başarıyla silindi.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/districts")
}

func (h *DashboardDistrictHandler) getFormData(c fiber.Ctx) map[string]string {
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
