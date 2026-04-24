package requests

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type BaseDistrictRequest struct {
	CountryID string `form:"country_id" validate:"required"`
	CityID    string `form:"city_id" validate:"required"`
	Name      string `form:"name" validate:"required,min=2,max=100"`
	IsActive  string `form:"is_active" validate:"required,oneof=true false"`
}

type ConvertedBaseDistrictRequest struct {
	CountryID *uint
	CityID    *uint
	Name      string
	IsActive  *bool
}

func (r *BaseDistrictRequest) Convert() ConvertedBaseDistrictRequest {
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
	var cityIDPtr *uint
	if r.CityID != "" {
		if val, err := strconv.ParseUint(r.CityID, 10, 32); err == nil {
			uintVal := uint(val)
			cityIDPtr = &uintVal
		}
	}
	return ConvertedBaseDistrictRequest{
		CountryID: countryIDPtr,
		CityID:     cityIDPtr,
		Name:       strings.TrimSpace(r.Name),
		IsActive:   isActivePtr,
	}
}

type CreateDistrictRequest struct {
	BaseDistrictRequest
}

type UpdateDistrictRequest struct {
	BaseDistrictRequest
}

type DistrictListRequest struct {
	CountryID string `query:"country_id"`
	CityID    string `query:"city_id"`
	Name      string `query:"name"`
	IsActive  string `query:"is_active" validate:"omitempty,oneof=true false"`
	SortBy    string `query:"sortBy" validate:"omitempty,oneof=id name created_at"`
	OrderBy   string `query:"orderBy" validate:"omitempty,oneof=asc desc"`
	Page      string `query:"page" validate:"omitempty,numeric,min=1"`
	PerPage   string `query:"perPage" validate:"omitempty,numeric,min=1,max=200"`
}

type DistrictListParams struct {
	CountryID *uint
	CityID    *uint
	Name      string
	IsActive  string
	SortBy    string
	OrderBy   string
	Page      int
	PerPage   int
}

func (r *DistrictListRequest) ToServiceParams() DistrictListParams {
	params := DistrictListParams{
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
	if r.CityID != "" {
		if val, err := strconv.ParseUint(r.CityID, 10, 32); err == nil {
			uintVal := uint(val)
			params.CityID = &uintVal
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

func (p *DistrictListParams) applyDefaults() {
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

func (p *DistrictListParams) CalculateOffset() int {
	if p.Page <= 0 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

func ParseAndValidateCreateDistrictRequest(c fiber.Ctx) (CreateDistrictRequest, map[string]string, error) {
	var req CreateDistrictRequest
	if err := c.Bind().Form(&req); err != nil {
		return req, make(map[string]string), errors.New("geçersiz istek formatı")
	}
	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		return req, GetDistrictValidationErrors(err), errors.New("lütfen formdaki hataları düzeltin")
	}
	return req, make(map[string]string), nil
}

func ParseAndValidateUpdateDistrictRequest(c fiber.Ctx) (UpdateDistrictRequest, map[string]string, error) {
	var req UpdateDistrictRequest
	if err := c.Bind().Form(&req); err != nil {
		return req, make(map[string]string), errors.New("geçersiz istek formatı")
	}
	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		return req, GetDistrictValidationErrors(err), errors.New("lütfen formdaki hataları düzeltin")
	}
	return req, make(map[string]string), nil
}

func ParseAndValidateDistrictList(c fiber.Ctx) (DistrictListParams, map[string]string, error) {
	var req DistrictListRequest
	if err := c.Bind().Query(&req); err != nil {
		return DistrictListParams{}, make(map[string]string), errors.New("geçersiz sorgu parametreleri")
	}
	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		return DistrictListParams{}, GetDistrictListValidationErrors(err), errors.New("lütfen filtreleri kontrol edin")
	}
	return req.ToServiceParams(), make(map[string]string), nil
}

func GetDistrictValidationErrors(err error) map[string]string {
	errorMessages := map[string]string{
		"CountryID_required": "Ülke seçilmelidir.",
		"CityID_required":    "Şehir seçilmelidir.",
		"Name_required":     "İlçe adı zorunludur.",
		"Name_min":          "İlçe adı en az 2 karakter olmalıdır.",
		"Name_max":          "İlçe adı en fazla 100 karakter olabilir.",
		"IsActive_required": "Durum seçilmelidir.",
		"IsActive_oneof":    "Geçerli bir durum seçiniz.",
	}
	return CommonValidationErrors(err, errorMessages)
}

func GetDistrictListValidationErrors(err error) map[string]string {
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
