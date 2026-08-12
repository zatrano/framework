package validation

import (
	"strings"

	"github.com/zatrano/framework/packages/flash"
	"github.com/zatrano/framework/packages/http"
)

const (
	errorsFlashKey      = "_validation_errors"
	errorBagsFlashKey   = "_validation_error_bags"
	defaultErrorBagName = "default"
)

// MessageBag is a view-friendly validation error container.
type MessageBag struct {
	messages map[string][]string
}

// NewMessageBag creates a message bag from errors.
func NewMessageBag(errs Errors) MessageBag {
	out := make(map[string][]string, len(errs))
	for field, msgs := range errs {
		cp := append([]string(nil), msgs...)
		out[field] = cp
	}
	return MessageBag{messages: out}
}

// Has reports whether the field has errors.
func (b MessageBag) Has(field string) bool {
	return len(b.messages[field]) > 0
}

// First returns the first error for a field.
func (b MessageBag) First(field string) string {
	if msgs := b.messages[field]; len(msgs) > 0 {
		return msgs[0]
	}
	return ""
}

// Get returns all errors for a field.
func (b MessageBag) Get(field string) []string {
	return append([]string(nil), b.messages[field]...)
}

// Any reports whether any errors exist.
func (b MessageBag) Any() bool {
	return len(b.messages) > 0
}

// All returns a copy of all errors.
func (b MessageBag) All() map[string][]string {
	out := make(map[string][]string, len(b.messages))
	for k, v := range b.messages {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// FlashErrors stores validation errors in the session flash.
// An optional bag name stores errors under a named bag (for @error('field', 'bag')).
func FlashErrors(req *http.Request, errs Errors, bagName ...string) {
	if req == nil || req.Session() == nil || len(errs) == 0 {
		return
	}
	bag := defaultErrorBagName
	if len(bagName) > 0 && strings.TrimSpace(bagName[0]) != "" {
		bag = strings.TrimSpace(bagName[0])
	}
	copyErrs := map[string][]string{}
	for field, msgs := range errs {
		copyErrs[field] = append([]string(nil), msgs...)
	}
	if bag == defaultErrorBagName {
		req.Session().Flash(errorsFlashKey, copyErrs)
		return
	}
	bags := cloneErrorBags(rawErrorBags(req.Session().Get(errorBagsFlashKey)))
	bags[bag] = copyErrs
	req.Session().Flash(errorBagsFlashKey, bags)
}

// ErrorsFromSession loads the default flashed validation errors.
func ErrorsFromSession(req *http.Request) MessageBag {
	if req == nil || req.Session() == nil {
		return NewMessageBag(nil)
	}
	return NewMessageBag(parseErrorsValue(req.Session().Get(errorsFlashKey)))
}

// ErrorsBagFromSession loads a named error bag from the session flash.
func ErrorsBagFromSession(req *http.Request, bag string) MessageBag {
	if bag == "" || bag == defaultErrorBagName {
		return ErrorsFromSession(req)
	}
	bags := ErrorBagsFromSession(req)
	if v, ok := bags[bag]; ok {
		if mb, ok := v.(MessageBag); ok {
			return mb
		}
		return NewMessageBag(parseErrorsValue(v))
	}
	return NewMessageBag(nil)
}

// ErrorBagsFromSession returns named bags as map[string]any (MessageBag values) for views.
func ErrorBagsFromSession(req *http.Request) map[string]any {
	out := map[string]any{}
	if req == nil || req.Session() == nil {
		return out
	}
	for name, errs := range rawErrorBags(req.Session().Get(errorBagsFlashKey)) {
		out[name] = NewMessageBag(Errors(errs))
	}
	return out
}

// RedirectBack flashes errors + input and redirects back.
func (v *Validator) RedirectBack(req *http.Request, fallback string, bag ...string) *http.Response {
	FlashErrors(req, v.Errors(), bag...)
	WithInput(req)
	return http.RedirectBack(req, fallback)
}

// WithInput stores request input (minus secrets) for the next request.
func WithInput(req *http.Request, input ...map[string]string) {
	if req == nil {
		return
	}
	data := req.Except("password", "password_confirmation", "current_password", "_token")
	if len(input) > 0 && input[0] != nil {
		data = input[0]
	}
	flash.Old(req, data)
}

func parseErrorsValue(raw any) Errors {
	switch v := raw.(type) {
	case Errors:
		return v
	case map[string][]string:
		return Errors(v)
	case map[string]any:
		errs := Errors{}
		for field, value := range v {
			switch msgs := value.(type) {
			case []string:
				errs[field] = msgs
			case []any:
				list := make([]string, 0, len(msgs))
				for _, item := range msgs {
					list = append(list, stringifyAny(item))
				}
				errs[field] = list
			case string:
				errs[field] = []string{msgs}
			}
		}
		return errs
	default:
		return nil
	}
}

func rawErrorBags(raw any) map[string]map[string][]string {
	switch v := raw.(type) {
	case map[string]map[string][]string:
		return v
	case map[string]any:
		out := map[string]map[string][]string{}
		for name, value := range v {
			errs := parseErrorsValue(value)
			if len(errs) > 0 {
				out[name] = map[string][]string(errs)
			}
		}
		return out
	default:
		return map[string]map[string][]string{}
	}
}

func cloneErrorBags(in map[string]map[string][]string) map[string]map[string][]string {
	out := make(map[string]map[string][]string, len(in))
	for name, errs := range in {
		cp := make(map[string][]string, len(errs))
		for field, msgs := range errs {
			cp[field] = append([]string(nil), msgs...)
		}
		out[name] = cp
	}
	return out
}

func stringifyAny(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
