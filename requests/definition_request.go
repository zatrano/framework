package requests

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type BaseDefinitionRequest struct {
	Key         string `form:"key" validate:"required,min=2,max=120,definition_key"`
	Value       string `form:"value" validate:"max=50000"`
	Description string `form:"description" validate:"max=255"`
	IsActive    string `form:"is_active" validate:"required,oneof=true false"`
}

type ConvertedBaseDefinitionRequest struct {
	Key         string
	Value       string
	Description string
	IsActive    *bool
}

func (r *BaseDefinitionRequest) Convert() ConvertedBaseDefinitionRequest {
	var isActivePtr *bool
	if r.IsActive != "" {
		val := r.IsActive == "true"
		isActivePtr = &val
	}
	return ConvertedBaseDefinitionRequest{
		Key:         strings.TrimSpace(strings.ToLower(r.Key)),
		Value:       r.Value,
		Description: strings.TrimSpace(r.Description),
		IsActive:    isActivePtr,
	}
}

type CreateDefinitionRequest struct {
	BaseDefinitionRequest
}

type UpdateDefinitionRequest struct {
	BaseDefinitionRequest
}

type DefinitionListRequest struct {
	Key      string `query:"key"`
	IsActive string `query:"is_active" validate:"omitempty,oneof=true false"`
	SortBy   string `query:"sortBy" validate:"omitempty,oneof=id key created_at"`
	OrderBy  string `query:"orderBy" validate:"omitempty,oneof=asc desc"`
	Page     string `query:"page" validate:"omitempty,numeric,min=1"`
	PerPage  string `query:"perPage" validate:"omitempty,numeric,min=1,max=200"`
}

type DefinitionListParams struct {
	Key      string
	IsActive string
	SortBy   string
	OrderBy  string
	Page     int
	PerPage  int
}

func (r *DefinitionListRequest) ToServiceParams() DefinitionListParams {
	params := DefinitionListParams{
		Key:      strings.TrimSpace(r.Key),
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

func (p *DefinitionListParams) applyDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = 20
	}
	if p.SortBy == "" {
		p.SortBy = "key"
	}
	if p.OrderBy == "" {
		p.OrderBy = "asc"
	}
}

func (p *DefinitionListParams) CalculateOffset() int {
	if p.Page <= 0 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

func definitionKeyValidator(fl validator.FieldLevel) bool {
	s := strings.TrimSpace(strings.ToLower(fl.Field().String()))
	if len(s) < 2 {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	if s[0] == '_' || (s[0] >= '0' && s[0] <= '9') {
		return false
	}
	return true
}

func ParseAndValidateCreateDefinitionRequest(c fiber.Ctx) (CreateDefinitionRequest, map[string]string, error) {
	var req CreateDefinitionRequest
	if err := c.Bind().Form(&req); err != nil {
		return req, make(map[string]string), errors.New("geçersiz istek formatı")
	}
	v := validator.New()
	_ = v.RegisterValidation("definition_key", definitionKeyValidator)
	if err := v.Struct(&req); err != nil {
		return req, GetDefinitionValidationErrors(err), errors.New("lütfen formdaki hataları düzeltin")
	}
	return req, make(map[string]string), nil
}

func ParseAndValidateUpdateDefinitionRequest(c fiber.Ctx) (UpdateDefinitionRequest, map[string]string, error) {
	var req UpdateDefinitionRequest
	if err := c.Bind().Form(&req); err != nil {
		return req, make(map[string]string), errors.New("geçersiz istek formatı")
	}
	v := validator.New()
	_ = v.RegisterValidation("definition_key", definitionKeyValidator)
	if err := v.Struct(&req); err != nil {
		return req, GetDefinitionValidationErrors(err), errors.New("lütfen formdaki hataları düzeltin")
	}
	return req, make(map[string]string), nil
}

func ParseAndValidateDefinitionList(c fiber.Ctx) (DefinitionListParams, map[string]string, error) {
	var req DefinitionListRequest
	if err := c.Bind().Query(&req); err != nil {
		return DefinitionListParams{}, make(map[string]string), errors.New("geçersiz sorgu parametreleri")
	}
	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		return DefinitionListParams{}, GetDefinitionListValidationErrors(err), errors.New("lütfen filtreleri kontrol edin")
	}
	return req.ToServiceParams(), make(map[string]string), nil
}

func GetDefinitionValidationErrors(err error) map[string]string {
	errorMessages := map[string]string{
		"Key_required":       "Anahtar zorunludur.",
		"Key_min":            "Anahtar en az 2 karakter olmalıdır.",
		"Key_max":            "Anahtar en fazla 120 karakter olabilir.",
		"Key_definition_key": "Anahtar yalnızca küçük harf, rakam ve alt çizgi içerebilir; harf veya rakamla başlamalıdır.",
		"Value_max":          "Değer çok uzun.",
		"Description_max":    "Açıklama en fazla 255 karakter olabilir.",
		"IsActive_required":  "Durum seçilmelidir.",
		"IsActive_oneof":     "Geçerli bir durum seçiniz.",
	}
	return CommonValidationErrors(err, errorMessages)
}

func GetDefinitionListValidationErrors(err error) map[string]string {
	errorMessages := map[string]string{
		"IsActive_oneof":  "Durum sadece 'true' veya 'false' olabilir.",
		"SortBy_oneof":    "Sıralama alanı sadece 'id', 'key' veya 'created_at' olabilir.",
		"OrderBy_oneof":   "Sıralama yönü sadece 'asc' veya 'desc' olabilir.",
		"Page_numeric":    "Sayfa numarası sayı olmalıdır.",
		"Page_min":        "Sayfa numarası en az 1 olmalıdır.",
		"PerPage_numeric": "Sayfa başı kayıt sayısı sayı olmalıdır.",
		"PerPage_min":     "Sayfa başı kayıt en az 1 olmalıdır.",
		"PerPage_max":     "Sayfa başı kayıt en fazla 200 olmalıdır.",
	}
	return CommonValidationErrors(err, errorMessages)
}
