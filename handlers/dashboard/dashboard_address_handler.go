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

type DashboardAddressHandler struct {
	addressService  services.IAddressService
	districtService services.IDistrictService
	cityService     services.ICityService
	countryService  services.ICountryService
}

func NewDashboardAddressHandler(
	addressService services.IAddressService,
	districtService services.IDistrictService,
	cityService services.ICityService,
	countryService services.ICountryService,
) *DashboardAddressHandler {
	return &DashboardAddressHandler{
		addressService:  addressService,
		districtService: districtService,
		cityService:     cityService,
		countryService:  countryService,
	}
}

func (h *DashboardAddressHandler) loadLookups(ctx fiber.Ctx) ([]models.Country, []models.City, []models.District) {
	countries := []models.Country{}
	if cr, _ := h.countryService.GetAllCountries(ctx.Context(), requests.CountryListParams{Page: 1, PerPage: 1000, SortBy: "name", OrderBy: "asc"}); cr != nil {
		if data, ok := cr.Data.([]models.Country); ok {
			countries = data
		}
	}
	cities := []models.City{}
	if cr, _ := h.cityService.GetAllCities(ctx.Context(), requests.CityListParams{Page: 1, PerPage: 5000, SortBy: "name", OrderBy: "asc"}); cr != nil {
		if data, ok := cr.Data.([]models.City); ok {
			cities = data
		}
	}
	districts := []models.District{}
	if dr, _ := h.districtService.GetAllDistricts(ctx.Context(), requests.DistrictListParams{Page: 1, PerPage: 10000, SortBy: "name", OrderBy: "asc"}); dr != nil {
		if data, ok := dr.Data.([]models.District); ok {
			districts = data
		}
	}
	return countries, cities, districts
}

func (h *DashboardAddressHandler) ListAddresses(c fiber.Ctx) error {
	params, fieldErrors, err := requests.ParseAndValidateAddressList(c)
	if err != nil {
		countries, cities, districts := h.loadLookups(c)
		renderData := fiber.Map{
			"Title":            "Adresler",
			"ValidationErrors": fieldErrors,
			"Params":           params,
			"Result": &queryparams.PaginatedResult{
				Data: []models.Address{},
				Meta: queryparams.PaginationMeta{CurrentPage: params.Page, PerPage: params.PerPage, TotalItems: 0, TotalPages: 0},
			},
			"Countries": countries,
			"Cities":    cities,
			"Districts": districts,
		}
		return renderer.Render(c, "dashboard/addresses/list", "layouts/app", renderData, http.StatusBadRequest)
	}

	paginatedResult, err := h.addressService.GetAllAddresses(c.Context(), params)
	countries, cities, districts := h.loadLookups(c)

	renderData := fiber.Map{
		"Title":     "Adresler",
		"Result":    paginatedResult,
		"Params":    params,
		"Countries": countries,
		"Cities":    cities,
		"Districts": districts,
	}
	if err != nil {
		renderData[renderer.FlashErrorKeyView] = "Adresler getirilirken bir hata oluştu."
		renderData["Result"] = &queryparams.PaginatedResult{
			Data: []models.Address{},
			Meta: queryparams.PaginationMeta{CurrentPage: params.Page, PerPage: params.PerPage, TotalItems: 0, TotalPages: 0},
		}
	}
	return renderer.Render(c, "dashboard/addresses/list", "layouts/app", renderData, http.StatusOK)
}

func (h *DashboardAddressHandler) ShowCreateAddress(c fiber.Ctx) error {
	countries, cities, districts := h.loadLookups(c)
	return renderer.Render(c, "dashboard/addresses/create", "layouts/app", fiber.Map{
		"Title":     "Yeni Adres Ekle",
		"Countries": countries,
		"Cities":    cities,
		"Districts": districts,
	})
}

func (h *DashboardAddressHandler) CreateAddress(c fiber.Ctx) error {
	formData := h.getFormData(c)
	req, fieldErrors, err := requests.ParseAndValidateCreateAddressRequest(c)
	if err != nil {
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/dashboard/addresses/create")
	}
	if err := h.addressService.CreateAddress(c.Context(), req); err != nil {
		formflash.SetData(c, formData)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Adres oluşturulamadı: "+err.Error())
		return c.Redirect().To("/dashboard/addresses/create")
	}
	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Adres başarıyla oluşturuldu.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/addresses")
}

func (h *DashboardAddressHandler) ShowUpdateAddress(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString("Geçersiz adres ID")
	}
	addr, err := h.addressService.GetAddressByID(c.Context(), uint(id))
	if err != nil {
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Adres bulunamadı.")
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/addresses")
	}
	countries, cities, districts := h.loadLookups(c)
	return renderer.Render(c, "dashboard/addresses/update", "layouts/app", fiber.Map{
		"Title":     "Adres Düzenle",
		"Address":   addr,
		"Countries": countries,
		"Cities":    cities,
		"Districts": districts,
	})
}

func (h *DashboardAddressHandler) UpdateAddress(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz adres ID")
	}
	formData := h.getFormData(c)
	req, fieldErrors, err := requests.ParseAndValidateUpdateAddressRequest(c)
	if err != nil {
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/dashboard/addresses/update/" + c.Params("id"))
	}
	if err := h.addressService.UpdateAddress(c.Context(), uint(id), req); err != nil {
		formflash.SetData(c, formData)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Adres güncellenemedi: "+err.Error())
		return c.Redirect().To("/dashboard/addresses/update/" + c.Params("id"))
	}
	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Adres başarıyla güncellendi.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/addresses")
}

func (h *DashboardAddressHandler) DeleteAddress(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz adres ID")
	}
	if err := h.addressService.DeleteAddress(c.Context(), uint(id)); err != nil {
		errMsg := "Adres silinemedi: " + err.Error()
		if strings.Contains(c.Get("Accept"), "application/json") {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": errMsg})
		}
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, errMsg)
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/addresses")
	}
	if strings.Contains(c.Get("Accept"), "application/json") {
		return c.JSON(fiber.Map{"message": "Adres başarıyla silindi."})
	}
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Adres başarıyla silindi.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/addresses")
}

func (h *DashboardAddressHandler) getFormData(c fiber.Ctx) map[string]string {
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
