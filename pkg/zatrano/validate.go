package zatrano

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/framework/pkg/validation"
)

// Validate parses the request body into T, runs struct-tag validation,
// and sends a 422 JSON response on failure. Convenience re-export of validation.Validate[T].
//
// Usage in handlers:
//
//	func (h *Handler) Create(c fiber.Ctx) error {
//	    req, err := zatrano.Validate[CreateRequest](c)
//	    if err != nil { return err }
//	    // req is valid — use it
//	}
func Validate[T any](c fiber.Ctx) (T, error) {
	return validation.Validate[T](c)
}

// RegisterRule registers a custom validator tag on the global validation engine.
//
// Example:
//
//	zatrano.RegisterRule("tc_no", func(fl validator.FieldLevel) bool {
//	    return len(fl.Field().String()) == 11
//	})
func RegisterRule(tag string, fn validator.Func) error {
	return validation.RegisterRule(tag, fn)
}

// RegisterRuleWithMessage registers a custom tag with an i18n message key.
func RegisterRuleWithMessage(tag string, fn validator.Func, messageKey string) error {
	return validation.RegisterRuleWithMessage(tag, fn, messageKey)
}

// ValidationEngine returns the underlying validation.Engine for advanced usage.
func ValidationEngine() *validation.Engine {
	return validation.Default()
}
