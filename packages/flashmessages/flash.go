package flashmessages

import (
	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/configs/sessionconfig"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type UtilError string

func (e UtilError) Error() string { return string(e) }

const (
	ErrSessionStartFailed UtilError = "session başlatılamadı"
	ErrSessionSaveFailed  UtilError = "session kaydedilemedi"
)

const (
	FlashSuccessKey = "flash_success_message"
	FlashErrorKey   = "flash_error_message"
)

type FlashMessagesData struct {
	Success string
	Error   string
}

// v3: parametre tipi fiber.Ctx (concrete type)
func SetFlashMessage(c fiber.Ctx, key string, message string) error {
	sess, err := sessionconfig.SessionStart(c)
	if err != nil {
		logconfig.Log.Error("Flash mesajı için session başlatılamadı", zap.Error(err))
		return ErrSessionStartFailed
	}
	sess.Set(key, message)
	if err := sess.Save(); err != nil {
		logconfig.Log.Error("Flash mesajı için session kaydedilemedi", zap.Error(err))
		return ErrSessionSaveFailed
	}
	return nil
}

func GetFlashMessages(c fiber.Ctx) (FlashMessagesData, error) {
	messages := FlashMessagesData{}
	sess, err := sessionconfig.SessionStart(c)
	if err != nil {
		logconfig.Log.Error("Flash mesajları alınırken session başlatılamadı", zap.Error(err))
		return messages, ErrSessionStartFailed
	}

	var sessionNeedsSave bool

	if success := sess.Get(FlashSuccessKey); success != nil {
		if msg, ok := success.(string); ok {
			messages.Success = msg
			sess.Delete(FlashSuccessKey)
			sessionNeedsSave = true
		}
	}
	if errorFlash := sess.Get(FlashErrorKey); errorFlash != nil {
		if msg, ok := errorFlash.(string); ok {
			messages.Error = msg
			sess.Delete(FlashErrorKey)
			sessionNeedsSave = true
		}
	}

	if sessionNeedsSave {
		if err := sess.Save(); err != nil {
			logconfig.Log.Error("Flash mesajları alındıktan sonra session kaydedilemedi", zap.Error(err))
			return messages, ErrSessionSaveFailed
		}
	}
	return messages, nil
}
