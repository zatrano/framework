package middlewares

import (
	"github.com/zatrano/framework/configs/sessionconfig"

	"github.com/gofiber/fiber/v3"
)

// v3: fiber.Ctx artık interface değil; tüm imzalar güncellendi
func SessionMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		sess, err := sessionconfig.SessionStart(c)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Session başlatılamadı")
		}
		c.Locals("session", sess)
		return c.Next()
	}
}
