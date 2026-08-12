package validation

import (
	"strings"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

const precognitionHeader = "Precognition"
const precognitionValidateOnly = "Precognition-Validate-Only"

// IsPrecognitive reports whether the request is a precognition request.
func IsPrecognitive(req *http.Request) bool {
	value := strings.ToLower(req.Header(precognitionHeader))
	return value == "true" || value == "1"
}

// ValidateOnlyFields returns fields listed in Precognition-Validate-Only.
func ValidateOnlyFields(req *http.Request) []string {
	header := strings.TrimSpace(req.Header(precognitionValidateOnly))
	if header == "" {
		return nil
	}
	parts := strings.Split(header, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// FilterRules keeps only selected fields when validate-only is present.
func FilterRules(rules map[string]string, only []string) map[string]string {
	if len(only) == 0 {
		return rules
	}
	filtered := make(map[string]string, len(only))
	for _, field := range only {
		if rule, ok := rules[field]; ok {
			filtered[field] = rule
		}
	}
	return filtered
}

// Precognition runs validation for precognitive requests and short-circuits.
// On success returns 204. On failure returns 422 with errors.
func Precognition(form FormRequest) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if !IsPrecognitive(req) {
				return next(req)
			}

			if !form.Authorize(req) {
				return ResponseFor(req, FailedAuthorization{})
			}

			rules := FilterRules(form.Rules(), ValidateOnlyFields(req))
			validator := Make(req.All(), rules)
			if messages := form.Messages(); messages != nil {
				validator.SetMessages(messages)
			}
			if validator.Fails() {
				return ResponseFor(req, ValidationException{Errors: validator.Errors()})
			}

			resp := http.NoContent()
			resp.Header("Precognition-Success", "true")
			return resp
		}
	}
}

// WithForm validates a form request before invoking the handler.
// Precognitive requests are handled automatically.
func WithForm(form FormRequest, handler func(req *http.Request, data map[string]string) *http.Response) routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		if IsPrecognitive(req) {
			return Precognition(form)(func(req *http.Request) *http.Response {
				return http.NoContent()
			})(req)
		}

		data, err := ValidateForm(req, form)
		if err != nil {
			return ResponseFor(req, err)
		}
		return handler(req, data)
	}
}
