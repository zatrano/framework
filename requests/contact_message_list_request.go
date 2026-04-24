package requests

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type ContactMessageListRequest struct {
	Name    string `query:"name"`
	Email   string `query:"email"`
	Subject string `query:"subject"`
	Unread  string `query:"unread"`
	SortBy  string `query:"sortBy"`
	OrderBy string `query:"orderBy"`
	Page    string `query:"page"`
	PerPage string `query:"perPage"`
}

type ContactMessageListParams struct {
	Name    string
	Email   string
	Subject string
	Unread  string
	SortBy  string
	OrderBy string
	Page    int
	PerPage int
}

func (r *ContactMessageListRequest) ToServiceParams() ContactMessageListParams {
	params := ContactMessageListParams{
		Name:    strings.TrimSpace(r.Name),
		Email:   strings.TrimSpace(r.Email),
		Subject: strings.TrimSpace(r.Subject),
		Unread:  strings.TrimSpace(r.Unread),
		SortBy:  strings.TrimSpace(r.SortBy),
		OrderBy: strings.TrimSpace(r.OrderBy),
	}
	if r.Page != "" {
		if p, err := strconv.Atoi(r.Page); err == nil && p > 0 {
			params.Page = p
		}
	}
	if r.PerPage != "" {
		if pp, err := strconv.Atoi(r.PerPage); err == nil && pp > 0 && pp <= 200 {
			params.PerPage = pp
		}
	}
	params.ApplyDefaults()
	return params
}

func (p *ContactMessageListParams) ApplyDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 200 {
		p.PerPage = 20
	}
	if p.SortBy == "" {
		p.SortBy = "created_at"
	}
	if p.OrderBy == "" {
		p.OrderBy = "desc"
	}
}

func (p *ContactMessageListParams) CalculateOffset() int {
	if p.Page <= 0 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

// v3: c.Bind().Query(&req)
func ParseAndValidateContactMessageList(c fiber.Ctx) (ContactMessageListParams, map[string]string, error) {
	var req ContactMessageListRequest
	if err := c.Bind().Query(&req); err != nil {
		return ContactMessageListParams{}, nil, err
	}
	return req.ToServiceParams(), nil, nil
}
