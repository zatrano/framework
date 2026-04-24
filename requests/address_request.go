package requests

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type BaseAddressRequest struct {
	CountryID  string `form:"country_id" validate:"required"`
	CityID     string `form:"city_id" validate:"required"`
	DistrictID string `form:"district_id" validate:"required"`
	Detail     string `form:"detail" validate:"required,min=2,max=255"`
	IsActive   string `form:"is_active" validate:"required,oneof=true false"`
}

type ConvertedBaseAddressRequest struct {
	CountryID  *uint
	CityID     *uint
	DistrictID *uint
	Detail     string
	IsActive   *bool
}

func (r *BaseAddressRequest) Convert() ConvertedBaseAddressRequest {
	var isActivePtr *bool
	if r.IsActive != "" {
		val := r.IsActive == "true"
		isActivePtr = &val
	}
	var countryIDPtr *uint
	if r.CountryID != "" {
		if val, err := strconv.ParseUint(r.CountryID, 10, 32); err == nil {
			u := uint(val)
			countryIDPtr = &u
		}
	}
	var cityIDPtr *uint
	if r.CityID != "" {
		if val, err := strconv.ParseUint(r.CityID, 10, 32); err == nil {
			u := uint(val)
			cityIDPtr = &u
		}
	}
	var districtIDPtr *uint
	if r.DistrictID != "" {
		if val, err := strconv.ParseUint(r.DistrictID, 10, 32); err == nil {
			u := uint(val)
			districtIDPtr = &u
		}
	}
	return ConvertedBaseAddressRequest{
		CountryID:  countryIDPtr,
		CityID:     cityIDPtr,
		DistrictID: districtIDPtr,
		Detail:     strings.TrimSpace(r.Detail),
		IsActive:   isActivePtr,
	}
}

type CreateAddressRequest struct {
	BaseAddressRequest
}

type UpdateAddressRequest struct {
	BaseAddressRequest
}

type AddressListRequest struct {
	CountryID  string `query:"country_id"`
	CityID     string `query:"city_id"`
	DistrictID string `query:"district_id"`
	Detail     string `query:"detail"`
	IsActive   string `query:"is_active" validate:"omitempty,oneof=true false"`
	SortBy     string `query:"sortBy" validate:"omitempty,oneof=id detail created_at"`
	OrderBy    string `query:"orderBy" validate:"omitempty,oneof=asc desc"`
	Page       string `query:"page" validate:"omitempty,numeric,min=1"`
	PerPage    string `query:"perPage" validate:"omitempty,numeric,min=1,max=200"`
}

type AddressListParams struct {
	CountryID  *uint
	CityID     *uint
	DistrictID *uint
	Detail     string
	IsActive   string
	SortBy     string
	OrderBy    string
	Page       int
	PerPage    int
}

func (r *AddressListRequest) ToServiceParams() AddressListParams {
	params := AddressListParams{
		Detail:   strings.TrimSpace(r.Detail),
		IsActive: strings.TrimSpace(r.IsActive),
		SortBy:   strings.TrimSpace(r.SortBy),
		OrderBy:  strings.TrimSpace(r.OrderBy),
	}
	if r.CountryID != "" {
		if val, err := strconv.ParseUint(r.CountryID, 10, 32); err == nil {
			u := uint(val)
			params.CountryID = &u
		}
	}
	if r.CityID != "" {
		if val, err := strconv.ParseUint(r.CityID, 10, 32); err == nil {
			u := uint(val)
			params.CityID = &u
		}
	}
	if r.DistrictID != "" {
		if val, err := strconv.ParseUint(r.DistrictID, 10, 32); err == nil {
			u := uint(val)
			params.DistrictID = &u
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

func (p *AddressListParams) applyDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = 20
	}
	if p.SortBy == "" {
		p.SortBy = "id"
	}
	if p.OrderBy == "" {
		p.OrderBy = "desc"
	}
}

func (p *AddressListParams) CalculateOffset() int {
	if p.Page <= 0 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

func ParseAndValidateCreateAddressRequest(c fiber.Ctx) (CreateAddressRequest, map[string]string, error) {
	var req CreateAddressRequest
	if err := c.Bind().Form(&req); err != nil {
		return req, make(map[string]string), errors.New("geçersiz istek formatı")
	}
	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		return req, GetAddressValidationErrors(err), errors.New("lütfen formdaki hataları düzeltin")
	}
	return req, make(map[string]string), nil
}

func ParseAndValidateUpdateAddressRequest(c fiber.Ctx) (UpdateAddressRequest, map[string]string, error) {
	var req UpdateAddressRequest
	if err := c.Bind().Form(&req); err != nil {
		return req, make(map[string]string), errors.New("geçersiz istek formatı")
	}
	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		return req, GetAddressValidationErrors(err), errors.New("lütfen formdaki hataları düzeltin")
	}
	return req, make(map[string]string), nil
}

func ParseAndValidateAddressList(c fiber.Ctx) (AddressListParams, map[string]string, error) {
	var req AddressListRequest
	if err := c.Bind().Query(&req); err != nil {
		return AddressListParams{}, make(map[string]string), errors.New("geçersiz sorgu parametreleri")
	}
	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		return AddressListParams{}, GetAddressListValidationErrors(err), errors.New("lütfen filtreleri kontrol edin")
	}
	return req.ToServiceParams(), make(map[string]string), nil
}

func GetAddressValidationErrors(err error) map[string]string {
	errorMessages := map[string]string{
		"CountryID_required":  "Ülke seçilmelidir.",
		"CityID_required":     "Şehir seçilmelidir.",
		"DistrictID_required": "İlçe seçilmelidir.",
		"Detail_required":     "Adres detayı zorunludur.",
		"Detail_min":          "Adres detayı en az 2 karakter olmalıdır.",
		"Detail_max":          "Adres detayı en fazla 255 karakter olabilir.",
		"IsActive_required":   "Durum seçilmelidir.",
		"IsActive_oneof":      "Geçerli bir durum seçiniz.",
	}
	return CommonValidationErrors(err, errorMessages)
}

func GetAddressListValidationErrors(err error) map[string]string {
	errorMessages := map[string]string{
		"IsActive_oneof":  "Durum sadece 'true' veya 'false' olabilir.",
		"SortBy_oneof":    "Sıralama alanı sadece 'id', 'detail' veya 'created_at' olabilir.",
		"OrderBy_oneof":   "Sıralama yönü sadece 'asc' veya 'desc' olabilir.",
		"Page_numeric":    "Sayfa numarası sayı olmalıdır.",
		"Page_min":        "Sayfa numarası en az 1 olmalıdır.",
		"PerPage_numeric": "Sayfa başı kayıt sayısı sayı olmalıdır.",
		"PerPage_min":     "Sayfa başı kayıt en az 1 olmalıdır.",
		"PerPage_max":     "Sayfa başı kayıt en fazla 200 olmalıdır.",
	}
	return CommonValidationErrors(err, errorMessages)
}
