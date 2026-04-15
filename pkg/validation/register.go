package validation

import (
	"github.com/go-playground/validator/v10"
)

// RegisterRule adds a custom validation tag to the global engine.
//
// Example:
//
//	validation.RegisterRule("tc_no", func(fl validator.FieldLevel) bool {
//	    return len(fl.Field().String()) == 11
//	})
//
// Then use it in struct tags:
//
//	type Request struct {
//	    TCNO string `json:"tc_no" validate:"required,tc_no"`
//	}
func RegisterRule(tag string, fn validator.Func) error {
	return Default().v.RegisterValidation(tag, fn)
}

// RegisterRuleWithMessage adds a custom validation tag and associates an i18n
// message key for translation. The key is looked up from the i18n bundle when
// producing error messages (e.g. "validation.tc_no" → "Geçerli bir TC kimlik numarası olmalıdır").
func RegisterRuleWithMessage(tag string, fn validator.Func, messageKey string) error {
	e := Default()
	if err := e.v.RegisterValidation(tag, fn); err != nil {
		return err
	}
	e.mu.Lock()
	e.messageKeys[tag] = messageKey
	e.mu.Unlock()
	return nil
}
