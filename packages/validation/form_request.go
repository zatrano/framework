package validation

import (
	"fmt"

	"github.com/zatrano/framework/packages/http"
)

// FormRequest defines a validated request contract.
type FormRequest interface {
	Rules() map[string]string
	Authorize(req *http.Request) bool
	Messages() map[string]string
}

// AttributesAware optionally supplies human-friendly field names.
type AttributesAware interface {
	Attributes() map[string]string
}

// ErrorBagAware optionally stores failures under a named error bag.
type ErrorBagAware interface {
	ErrorBag() string
}

// PreparesValidation optionally mutates the request before rules run.
type PreparesValidation interface {
	PrepareForValidation(req *http.Request)
}

// Base can be embedded for default authorize/messages behavior.
type Base struct{}

// Authorize allows the request by default.
func (Base) Authorize(req *http.Request) bool { return true }

// Messages returns custom messages (none by default).
func (Base) Messages() map[string]string { return nil }

// Attributes returns custom attribute names (none by default).
func (Base) Attributes() map[string]string { return nil }

// ErrorBag returns the default error bag.
func (Base) ErrorBag() string { return "" }

// PrepareForValidation is a no-op by default.
func (Base) PrepareForValidation(req *http.Request) {}

// FailedAuthorization is returned when Authorize fails.
type FailedAuthorization struct{}

func (FailedAuthorization) Error() string { return "This action is unauthorized." }

// ValidationException carries validation errors and an optional named bag.
type ValidationException struct {
	Errors Errors
	Bag    string
}

func (e ValidationException) Error() string { return "validation failed" }

// ValidateForm validates a form request against the HTTP request.
func ValidateForm(req *http.Request, form FormRequest) (map[string]string, error) {
	if !form.Authorize(req) {
		return nil, FailedAuthorization{}
	}

	if p, ok := form.(PreparesValidation); ok {
		p.PrepareForValidation(req)
	}

	validator := Make(req.All(), form.Rules())
	if messages := form.Messages(); messages != nil {
		validator.SetMessages(messages)
	}
	if a, ok := form.(AttributesAware); ok {
		if attrs := a.Attributes(); attrs != nil {
			validator.SetAttributes(attrs)
		}
	}

	bag := ""
	if b, ok := form.(ErrorBagAware); ok {
		bag = b.ErrorBag()
	}

	if validator.Fails() {
		return nil, ValidationException{Errors: validator.Errors(), Bag: bag}
	}
	return validator.Validated()
}

// ValidateRequest validates request input against rules and returns validated fields.
func ValidateRequest(req *http.Request, rules map[string]string, messages ...map[string]string) (map[string]string, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	validator := Make(req.All(), rules)
	if len(messages) > 0 && messages[0] != nil {
		validator.SetMessages(messages[0])
	}
	return validator.Validated()
}

// ResponseFor converts form request failures into HTTP responses.
func ResponseFor(req *http.Request, err error) *http.Response {
	switch e := err.(type) {
	case FailedAuthorization:
		if req != nil && req.WantsJSON() {
			return http.JSON(map[string]any{"message": e.Error()}).Status(403)
		}
		return http.Abort(403, e.Error())
	case ValidationException:
		if req != nil && (req.WantsJSON() || IsPrecognitive(req)) {
			return http.JSON(map[string]any{
				"message": "Validation failed",
				"errors":  e.Errors.Message(),
			}).Status(422)
		}
		if req != nil {
			FlashErrors(req, e.Errors, e.Bag)
			WithInput(req)
			return http.RedirectBack(req, "/")
		}
		return http.JSON(map[string]any{
			"message": "Validation failed",
			"errors":  e.Errors.Message(),
		}).Status(422)
	default:
		return http.JSON(map[string]any{"message": err.Error()}).Status(500)
	}
}
