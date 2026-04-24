package requests

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type BaseCityRequest struct {
	CountryID string `form:"country_id" validate:"required"`
	Name      string `form:"name" validate:"required,min=2,max=100"`
	IsActive  string `form:"is_active" validate:"required,oneof=true false"`
}

type ConvertedBaseCityRequest struct {
	CountryID *uint
	Name      string
	IsActive  *bool
}

func (r *BaseCityRequest) Convert() ConvertedBaseCityRequest {
	var isActivePtr *bool
	if r.IsActive != "" {
		val := r.IsActive == "true"
		isActivePtr = &val
	}
	var countryIDPtr *uint
	if r.CountryID != "" {
		if val, err := strconv.ParseUint(r.CountryID, 10, 32); err == nil {
			uintVal := uint(val)
			countryIDPtr = &uintVal
		}
	}
	return ConvertedBaseCityRequest{
		CountryID: countryIDPtr,
		Name:      strings.TrimSpace(r.Name),
		IsActive:  isActivePtr,
	}
}

type CreateCityRequest struct {
	BaseCityRequest
}

type UpdateCityRequest struct {
	BaseCityRequest
}

type CityListRequest struct {
	CountryID string `query:"country_id"`
	Name      string `query:"name"`
	IsActive  string `query:"is_active" validate:"omitempty,oneof=true false"`
	SortBy    string `query:"sortBy" validate:"omitempty,oneof=id name created_at"`
	OrderBy   string `query:"orderBy" validate:"omitempty,oneof=asc desc"`
	Page      string `query:"page" validate:"omitempty,numeric,min=1"`
	PerPage   string `query:"perPage" validate:"omitempty,numeric,min=1,max=200"`
}

type CityListParams struct {
	CountryID *uint
	Name      string
	IsActive  string
	SortBy    string
	OrderBy   string
	Page      int
	PerPage   int
}

func (r *CityListRequest) ToServiceParams() CityListParams {
	params := CityListParams{
		Name:     strings.TrimSpace(r.Name),
		IsActive: strings.TrimSpace(r.IsActive),
		SortBy:   strings.TrimSpace(r.SortBy),
		OrderBy:  strings.TrimSpace(r.OrderBy),
	}
	if r.CountryID != "" {
		if val, err := strconv.ParseUint(r.CountryID, 10, 32); err == nil {
			uintVal := uint(val)
			params.CountryID = &uintVal
		}
	}
	if r.Page != "" {
		if page, err := strconv.Atoi(r.Page); err == nil && page > 0 {
			params.Page = page
		}
	}
	if r.PerPage != "" {
		if perPage, err := strconv.Atoi(r.PerPage); err == nil && perPage > 0 {
			params.PerPage = perPage
		}
	}
	params.applyDefaults()
	return params
}

func (p *CityListParams) applyDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = 20
	}
	if p.SortBy == "" {
		p.SortBy = "name"
	}
	if p.OrderBy == "" {
		p.OrderBy = "asc"
	}
}

func (p *CityListParams) CalculateOffset() int {
	if p.Page <= 0 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

func ParseAndValidateCreateCityRequest(c fiber.Ctx) (CreateCityRequest, map[string]string, error) {
	var req CreateCityRequest
	if err := c.Bind().Form(&req); err != nil {
		return req, make(map[string]string), errors.New("geçersiz istek formatı")
	}
	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		return req, GetCityValidationErrors(err), errors.New("lütfen formdaki hataları düzeltin")
	}
	return req, make(map[string]string), nil
}

func ParseAndValidateUpdateCityRequest(c fiber.Ctx) (UpdateCityRequest, map[string]string, error) {
	var req UpdateCityRequest
	if err := c.Bind().Form(&req); err != nil {
		return req, make(map[string]string), errors.New("geçersiz istek formatı")
	}
	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		return req, GetCityValidationErrors(err), errors.New("lütfen formdaki hataları düzeltin")
	}
	return req, make(map[string]string), nil
}

func ParseAndValidateCityList(c fiber.Ctx) (CityListParams, map[string]string, error) {
	var req CityListRequest
	if err := c.Bind().Query(&req); err != nil {
		return CityListParams{}, make(map[string]string), errors.New("geçersiz sorgu parametreleri")
	}
	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		return CityListParams{}, GetCityListValidationErrors(err), errors.New("lütfen filtreleri kontrol edin")
	}
	return req.ToServiceParams(), make(map[string]string), nil
}

func GetCityValidationErrors(err error) map[string]string {
	errorMessages := map[string]string{
		"CountryID_required": "Ülke seçilmelidir.",
		"Name_required":      "Şehir adı zorunludur.",
		"Name_min":           "Şehir adı en az 2 karakter olmalıdır.",
		"Name_max":           "Şehir adı en fazla 100 karakter olabilir.",
		"IsActive_required":  "Durum seçilmelidir.",
		"IsActive_oneof":     "Geçerli bir durum seçiniz.",
	}
	return CommonValidationErrors(err, errorMessages)
}

func GetCityListValidationErrors(err error) map[string]string {
	errorMessages := map[string]string{
		"IsActive_oneof":  "Durum sadece 'true' veya 'false' olabilir.",
		"SortBy_oneof":    "Sıralama alanı sadece 'id', 'name' veya 'created_at' olabilir.",
		"OrderBy_oneof":   "Sıralama yönü sadece 'asc' veya 'desc' olabilir.",
		"Page_numeric":    "Sayfa numarası sayı olmalıdır.",
		"Page_min":        "Sayfa numarası en az 1 olmalıdır.",
		"PerPage_numeric": "Sayfa başı kayıt sayısı sayı olmalıdır.",
		"PerPage_min":     "Sayfa başı kayıt en az 1 olmalıdır.",
		"PerPage_max":     "Sayfa başı kayıt en fazla 200 olmalıdır.",
	}
	return CommonValidationErrors(err, errorMessages)
}
