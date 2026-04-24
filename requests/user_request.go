package requests

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type BaseUserRequest struct {
	Name              string `form:"name" validate:"required,min=3"`
	Email             string `form:"email" validate:"required,email"`
	IsActive          string `form:"is_active" validate:"required,oneof=true false"`
	UserTypeID        string `form:"user_type_id" validate:"required"`
	EmailVerified     string `form:"email_verified" validate:"omitempty,oneof=true false"`
	ResetToken        string `form:"reset_token"`
	VerificationToken string `form:"verification_token"`
	Provider          string `form:"provider"`
	ProviderID        string `form:"provider_id"`
}

type ConvertedBaseUserRequest struct {
	Name              string
	Email             string
	IsActive          *bool
	UserTypeID        *uint
	EmailVerified     *bool
	ResetToken        string
	VerificationToken string
	Provider          string
	ProviderID        string
}

func (r *BaseUserRequest) Convert() ConvertedBaseUserRequest {
	var isActivePtr *bool
	if r.IsActive != "" {
		val := r.IsActive == "true"
		isActivePtr = &val
	}
	var emailVerifiedPtr *bool
	if r.EmailVerified != "" {
		val := r.EmailVerified == "true"
		emailVerifiedPtr = &val
	}
	var userTypeIDPtr *uint
	if r.UserTypeID != "" {
		if val, err := strconv.ParseUint(r.UserTypeID, 10, 32); err == nil {
			uintVal := uint(val)
			userTypeIDPtr = &uintVal
		}
	}
	return ConvertedBaseUserRequest{
		Name:              strings.TrimSpace(r.Name),
		Email:             strings.ToLower(strings.TrimSpace(r.Email)),
		IsActive:          isActivePtr,
		UserTypeID:        userTypeIDPtr,
		EmailVerified:     emailVerifiedPtr,
		ResetToken:        r.ResetToken,
		VerificationToken: r.VerificationToken,
		Provider:          r.Provider,
		ProviderID:        r.ProviderID,
	}
}

type CreateUserRequest struct {
	BaseUserRequest
	Password        string `form:"password" validate:"required,min=8"`
	ConfirmPassword string `form:"confirm_password" validate:"required,eqfield=Password"`
}

type UpdateUserRequest struct {
	BaseUserRequest
	Password        string `form:"password" validate:"omitempty,min=8"`
	ConfirmPassword string `form:"confirm_password" validate:"omitempty,eqfield=Password"`
}

type UserListRequest struct {
	Name          string `query:"name"`
	Email         string `query:"email"`
	IsActive      string `query:"is_active" validate:"omitempty,oneof=true false"`
	EmailVerified string `query:"email_verified" validate:"omitempty,oneof=true false"`
	UserTypeID    string `query:"user_type_id"`
	SortBy     string `query:"sortBy" validate:"omitempty,oneof=id name email created_at"`
	OrderBy    string `query:"orderBy" validate:"omitempty,oneof=asc desc"`
	Page       string `query:"page"`
	PerPage    string `query:"perPage"`
}

type UserListParams struct {
	Name          string
	Email         string
	IsActive      string
	EmailVerified string
	UserTypeID    *uint
	SortBy        string
	OrderBy       string
	Page          int
	PerPage       int
}

func (r *UserListRequest) ToServiceParams() UserListParams {
	params := UserListParams{
		Name:          strings.TrimSpace(r.Name),
		Email:         strings.TrimSpace(r.Email),
		IsActive:      strings.TrimSpace(r.IsActive),
		EmailVerified: strings.TrimSpace(r.EmailVerified),
		SortBy:        strings.TrimSpace(r.SortBy),
		OrderBy:       strings.TrimSpace(r.OrderBy),
	}
	if r.UserTypeID != "" {
		if val, err := strconv.ParseUint(r.UserTypeID, 10, 32); err == nil {
			u := uint(val)
			params.UserTypeID = &u
		}
	}
	if r.Page != "" {
		if p, err := strconv.Atoi(r.Page); err == nil && p > 0 {
			params.Page = p
		}
	}
	if r.PerPage != "" {
		if pp, err := strconv.Atoi(r.PerPage); err == nil && pp > 0 {
			params.PerPage = pp
		}
	}
	params.applyDefaults()
	return params
}

func (p *UserListParams) applyDefaults() {
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

func (p *UserListParams) CalculateOffset() int {
	if p.Page <= 0 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

// v3: c.Bind().Form(&req)
func ParseAndValidateCreateUserRequest(c fiber.Ctx) (CreateUserRequest, map[string]string, error) {
	var req CreateUserRequest
	if err := c.Bind().Form(&req); err != nil {
		return req, make(map[string]string), errors.New("geçersiz istek formatı")
	}
	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		return req, GetUserValidationErrors(err), errors.New("lütfen formdaki hataları düzeltin")
	}
	return req, make(map[string]string), nil
}

func ParseAndValidateUpdateUserRequest(c fiber.Ctx) (UpdateUserRequest, map[string]string, error) {
	var req UpdateUserRequest
	if err := c.Bind().Form(&req); err != nil {
		return req, make(map[string]string), errors.New("geçersiz istek formatı")
	}
	validate := validator.New()
	if err := validate.Struct(&req); err != nil {
		return req, GetUserValidationErrors(err), errors.New("lütfen formdaki hataları düzeltin")
	}
	return req, make(map[string]string), nil
}

// v3: c.Bind().Query(&req)
func ParseAndValidateUserList(c fiber.Ctx) (UserListParams, map[string]string, error) {
	var req UserListRequest
	if err := c.Bind().Query(&req); err != nil {
		return UserListParams{}, make(map[string]string), errors.New("geçersiz sorgu parametreleri")
	}
	return req.ToServiceParams(), make(map[string]string), nil
}

func GetUserValidationErrors(err error) map[string]string {
	return CommonValidationErrors(err, map[string]string{
		"Name_required":            "İsim zorunludur.",
		"Name_min":                 "İsim en az 3 karakter olmalıdır.",
		"Email_required":           "E-posta zorunludur.",
		"Email_email":              "Geçerli bir e-posta adresi giriniz.",
		"IsActive_required":        "Durum seçilmelidir.",
		"IsActive_oneof":           "Geçerli bir durum seçiniz.",
		"UserTypeID_required":      "Kullanıcı tipi seçilmelidir.",
		"Password_required":        "Şifre zorunludur.",
		"Password_min":             "Şifre en az 8 karakter olmalıdır.",
		"ConfirmPassword_required": "Şifre onayı zorunludur.",
		"ConfirmPassword_eqfield":  "Şifreler eşleşmiyor.",
	})
}
