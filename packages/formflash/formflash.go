package formflash

import (
	"encoding/json"

	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/configs/sessionconfig"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type FormError string

func (e FormError) Error() string { return string(e) }

const (
	ErrSessionStartFailed FormError = "session başlatılamadı"
	ErrSessionSaveFailed  FormError = "session kaydedilemedi"
	ErrJSONMarshalFailed  FormError = "JSON dönüşümü başarısız"
)

const (
	FormDataKey         = "form_flash_data"
	ValidationErrorsKey = "form_validation_errors"
)

// v3: tüm fiber.Ctx parametreleri güncellendi
func SetData(c fiber.Ctx, data interface{}) error {
	sess, err := sessionconfig.SessionStart(c)
	if err != nil {
		logconfig.Log.Error("Form verisi için session başlatılamadı", zap.Error(err))
		return ErrSessionStartFailed
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		logconfig.Log.Error("Form verisi JSON'a çevrilemedi", zap.Error(err))
		return ErrJSONMarshalFailed
	}
	sess.Set(FormDataKey, string(jsonData))
	if err := sess.Save(); err != nil {
		logconfig.Log.Error("Form verisi için session kaydedilemedi", zap.Error(err))
		return ErrSessionSaveFailed
	}
	return nil
}

func GetData(c fiber.Ctx) (map[string]interface{}, error) {
	sess, err := sessionconfig.SessionStart(c)
	if err != nil {
		return nil, ErrSessionStartFailed
	}
	formData := make(map[string]interface{})
	if data := sess.Get(FormDataKey); data != nil {
		if jsonStr, ok := data.(string); ok {
			if err := json.Unmarshal([]byte(jsonStr), &formData); err != nil {
				logconfig.Log.Error("Form verisi parse edilemedi", zap.Error(err))
				return nil, FormError("form verisi parse edilemedi")
			}
			sess.Delete(FormDataKey)
			sess.Save()
		}
	}
	return formData, nil
}

func SetValidationErrors(c fiber.Ctx, errors map[string]string) error {
	sess, err := sessionconfig.SessionStart(c)
	if err != nil {
		logconfig.Log.Error("Validasyon hataları için session başlatılamadı", zap.Error(err))
		return ErrSessionStartFailed
	}
	jsonErrors, err := json.Marshal(errors)
	if err != nil {
		logconfig.Log.Error("Validasyon hataları JSON'a çevrilemedi", zap.Error(err))
		return ErrJSONMarshalFailed
	}
	sess.Set(ValidationErrorsKey, string(jsonErrors))
	if err := sess.Save(); err != nil {
		logconfig.Log.Error("Validasyon hataları için session kaydedilemedi", zap.Error(err))
		return ErrSessionSaveFailed
	}
	return nil
}

func GetValidationErrors(c fiber.Ctx) (map[string]string, error) {
	sess, err := sessionconfig.SessionStart(c)
	if err != nil {
		return nil, ErrSessionStartFailed
	}
	errors := make(map[string]string)
	if errorsData := sess.Get(ValidationErrorsKey); errorsData != nil {
		if jsonStr, ok := errorsData.(string); ok {
			if err := json.Unmarshal([]byte(jsonStr), &errors); err != nil {
				return nil, FormError("validasyon hataları parse edilemedi")
			}
			sess.Delete(ValidationErrorsKey)
			sess.Save()
		}
	}
	return errors, nil
}

func ClearData(c fiber.Ctx) error {
	sess, err := sessionconfig.SessionStart(c)
	if err != nil {
		return ErrSessionStartFailed
	}
	sess.Delete(FormDataKey)
	sess.Delete(ValidationErrorsKey)
	return sess.Save()
}
