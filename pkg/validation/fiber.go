package validation

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/framework/pkg/i18n"
)

// Validate parses the request body into T, runs struct-tag validation,
// and returns a 422 JSON response on failure. Usage:
//
//	req, err := validation.Validate[CreateUserRequest](c)
//	if err != nil { return err }
func Validate[T any](c fiber.Ctx) (T, error) {
	var req T

	if err := c.Bind().Body(&req); err != nil {
		return req, c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    422,
				"message": "invalid request body",
				"details": []FieldError{{
					Field:   "_body",
					Tag:     "parse",
					Message: err.Error(),
				}},
			},
		})
	}

	locale := i18n.Locale(c)
	if verr := Default().ValidateStruct(req, locale); verr != nil {
		return req, c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    422,
				"message": "validation failed",
				"details": verr.Errors,
			},
		})
	}

	return req, nil
}
