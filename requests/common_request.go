package requests

import (
	"math"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

func convertFieldToTemplateKey(field string) string {
	var result strings.Builder
	for i, r := range field {
		if i > 0 && unicode.IsUpper(r) {
			if i > 1 && unicode.IsLower(rune(field[i-1])) {
				result.WriteByte('_')
			} else if i < len(field)-1 && unicode.IsLower(rune(field[i+1])) {
				result.WriteByte('_')
			}
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

func CommonValidationErrors(err error, errorMessages map[string]string) map[string]string {
	errors := make(map[string]string)
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			field := e.Field()
			tag := e.Tag()
			fieldKey := convertFieldToTemplateKey(field)
			key := field + "_" + tag
			if msg, ok := errorMessages[key]; ok {
				errors[fieldKey] = msg
			} else {
				errors[fieldKey] = "Geçersiz değer."
			}
		}
	}
	return errors
}

type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

type PaginatedResult struct {
	Data interface{}    `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

func CalculateTotalPages(totalItems int64, perPage int) int {
	if perPage <= 0 {
		return 1
	}
	return int(math.Ceil(float64(totalItems) / float64(perPage)))
}

func CreatePaginatedResult(data interface{}, totalItems int64, currentPage, perPage int) *PaginatedResult {
	return &PaginatedResult{
		Data: data,
		Meta: PaginationMeta{
			CurrentPage: currentPage,
			PerPage:     perPage,
			TotalItems:  totalItems,
			TotalPages:  CalculateTotalPages(totalItems, perPage),
		},
	}
}
