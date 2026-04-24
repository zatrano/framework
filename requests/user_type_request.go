package requests

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type BaseUserTypeRequest struct {
	Name     string `form:"name" validate:"required,min=2"`
	IsActive string `form:"is_active" validate:"required,oneof=true false"`
}

type ConvertedBaseUserTypeRequest struct {
	Name     string
	IsActive *bool
}

func (r *BaseUserTypeRequest) Convert() ConvertedBaseUserTypeRequest {
	var isActivePtr *bool
	if r.IsActive != "" {
		val := r.IsActive == "true"
		isActivePtr = &val
	}

	return ConvertedBaseUserTypeRequest{
		Name:     r.Name,
		IsActive: isActivePtr,
	}
}

type UserTypeRequest struct {
	BaseUserTypeRequest
}

func ParseAndValidateUserTypeRequest(c fiber.Ctx) (UserTypeRequest, map[string]string, error) {
	var req UserTypeRequest

	if err := c.Bind().Form(&req); err != nil {
		return req, make(map[string]string), errors.New("geçersiz istek formatı")
	}

	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		validationErrors := GetUserTypeValidationErrors(err)
		return req, validationErrors, errors.New("lütfen formdaki hataları düzeltin")
	}

	return req, make(map[string]string), nil
}

type UserTypeListRequest struct {
	Name     string `query:"name"`
	IsActive string `query:"is_active" validate:"omitempty,oneof=true false"`
	SortBy   string `query:"sortBy" validate:"omitempty,oneof=id name"`
	OrderBy  string `query:"orderBy" validate:"omitempty,oneof=asc desc"`
	Page     string `query:"page" validate:"omitempty,numeric,min=1"`
	PerPage  string `query:"perPage" validate:"omitempty,numeric,min=1,max=200"`
}

type UserTypeListParams struct {
	Name     string
	IsActive string
	SortBy   string
	OrderBy  string
	Page     int
	PerPage  int
}

func (r *UserTypeListRequest) ToServiceParams() UserTypeListParams {
	params := UserTypeListParams{
		Name:     strings.TrimSpace(r.Name),
		IsActive: strings.TrimSpace(r.IsActive),
		SortBy:   strings.TrimSpace(r.SortBy),
		OrderBy:  strings.TrimSpace(r.OrderBy),
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

func (p *UserTypeListParams) applyDefaults() {
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

func (p *UserTypeListParams) CalculateOffset() int {
	if p.Page <= 0 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

func ParseAndValidateUserTypeList(c fiber.Ctx) (UserTypeListParams, map[string]string, error) {
	var req UserTypeListRequest

	if err := c.Bind().Query(&req); err != nil {
		return UserTypeListParams{}, make(map[string]string), errors.New("geçersiz sorgu parametreleri")
	}

	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		validationErrors := GetUserTypeListValidationErrors(err)
		return UserTypeListParams{}, validationErrors, errors.New("lütfen filtreleri kontrol edin")
	}

	return req.ToServiceParams(), make(map[string]string), nil
}

func GetUserTypeValidationErrors(err error) map[string]string {
	errorMessages := map[string]string{
		"Name_required":     "Kullanıcı tipi adı zorunludur.",
		"Name_min":          "Kullanıcı tipi adı en az 2 karakter olmalıdır.",
		"IsActive_required": "Kullanıcı tipi durumu seçilmelidir.",
		"IsActive_oneof":    "Geçerli bir durum seçiniz (Aktif/Pasif).",
	}

	return CommonValidationErrors(err, errorMessages)
}

func GetUserTypeListValidationErrors(err error) map[string]string {
	errorMessages := map[string]string{
		"IsActive_oneof":  "Durum sadece 'true' veya 'false' olabilir.",
		"SortBy_oneof":    "Sıralama alanı sadece 'id' veya 'name' olabilir.",
		"OrderBy_oneof":   "Sıralama yönü sadece 'asc' veya 'desc' olabilir.",
		"Page_numeric":    "Sayfa numarası sayı olmalıdır.",
		"Page_min":        "Sayfa numarası en az 1 olmalıdır.",
		"PerPage_numeric": "Sayfa başı kayıt sayısı sayı olmalıdır.",
		"PerPage_min":     "Sayfa başı kayıt en az 1 olmalıdır.",
		"PerPage_max":     "Sayfa başı kayıt en fazla 200 olmalıdır.",
	}

	return CommonValidationErrors(err, errorMessages)
}
