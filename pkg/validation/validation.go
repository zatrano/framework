package validation

import (
	"sync"

	"github.com/go-playground/validator/v10"

	"github.com/zatrano/framework/pkg/i18n"
)

// Engine wraps go-playground/validator with optional i18n message translation.
type Engine struct {
	v      *validator.Validate
	bundle *i18n.Bundle // nil when i18n is disabled

	mu          sync.RWMutex
	messageKeys map[string]string // tag → i18n key override (e.g. "tc_no" → "validation.tc_no")
}

var (
	mu            sync.RWMutex
	defaultEngine *Engine
)

// Init creates (or replaces) the global validation engine.
// Called from core.Bootstrap; bundle may be nil when i18n is disabled.
func Init(bundle *i18n.Bundle) {
	e := &Engine{
		v:           validator.New(validator.WithRequiredStructEnabled()),
		bundle:      bundle,
		messageKeys: make(map[string]string),
	}

	mu.Lock()
	defaultEngine = e
	mu.Unlock()
}

// Default returns the global Engine. Panics if Init was not called.
func Default() *Engine {
	mu.RLock()
	e := defaultEngine
	mu.RUnlock()
	if e == nil {
		panic("validation: engine not initialised — call validation.Init first")
	}
	return e
}

// Validator exposes the underlying go-playground/validator instance for advanced usage.
func (e *Engine) Validator() *validator.Validate {
	return e.v
}

// Bundle returns the i18n bundle (may be nil).
func (e *Engine) Bundle() *i18n.Bundle {
	return e.bundle
}

// ValidateStruct runs struct-tag validation and returns nil or a *ValidationError.
func (e *Engine) ValidateStruct(s any, locale string) *ValidationError {
	err := e.v.Struct(s)
	if err == nil {
		return nil
	}

	fieldErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		// Unexpected error type — wrap as single-field error.
		return &ValidationError{
			Errors: []FieldError{{
				Field:   "_",
				Tag:     "unknown",
				Message: err.Error(),
			}},
		}
	}

	out := make([]FieldError, 0, len(fieldErrors))
	for _, fe := range fieldErrors {
		out = append(out, e.translateFieldError(fe, locale))
	}
	return &ValidationError{Errors: out}
}

// translateFieldError converts a single validator.FieldError to a FieldError with i18n message.
func (e *Engine) translateFieldError(fe validator.FieldError, locale string) FieldError {
	tag := fe.Tag()
	msgKey := "validation." + tag

	// Check for custom message key overrides.
	e.mu.RLock()
	if override, ok := e.messageKeys[tag]; ok {
		msgKey = override
	}
	e.mu.RUnlock()

	msg := msgKey // fallback: raw key
	if e.bundle != nil {
		translated := e.bundle.T(locale, msgKey)
		if translated != msgKey {
			// Replace {{.Param}} placeholder with actual param.
			msg = replaceParam(translated, fe.Param())
		}
	}

	// If still the raw key, build a programmer-friendly fallback.
	if msg == msgKey {
		msg = fallbackMessage(fe)
	}

	return FieldError{
		Field:   fe.Field(),
		Tag:     tag,
		Value:   fe.Value(),
		Message: msg,
	}
}

// replaceParam replaces the {{.Param}} placeholder in translated messages.
func replaceParam(msg, param string) string {
	if param == "" {
		return msg
	}
	// Simple string replacement — avoids full template overhead.
	result := msg
	for _, placeholder := range []string{"{{.Param}}", "{{ .Param }}"} {
		result = stringReplace(result, placeholder, param)
	}
	return result
}

func stringReplace(s, old, new string) string {
	if old == "" {
		return s
	}
	n := 0
	i := 0
	for {
		j := indexOf(s[i:], old)
		if j < 0 {
			break
		}
		n++
		i += j + len(old)
	}
	if n == 0 {
		return s
	}
	buf := make([]byte, 0, len(s)+(len(new)-len(old))*n)
	i = 0
	for range n {
		j := indexOf(s[i:], old)
		buf = append(buf, s[i:i+j]...)
		buf = append(buf, new...)
		i += j + len(old)
	}
	buf = append(buf, s[i:]...)
	return string(buf)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// fallbackMessage builds a human-readable message when no i18n translation is available.
func fallbackMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must be at least " + fe.Param() + " characters"
	case "max":
		return "must be at most " + fe.Param() + " characters"
	case "gte":
		return "must be greater than or equal to " + fe.Param()
	case "lte":
		return "must be less than or equal to " + fe.Param()
	case "len":
		return "must be exactly " + fe.Param() + " characters"
	case "oneof":
		return "must be one of: " + fe.Param()
	default:
		msg := "failed on '" + fe.Tag() + "'"
		if fe.Param() != "" {
			msg += " (param: " + fe.Param() + ")"
		}
		return msg
	}
}
