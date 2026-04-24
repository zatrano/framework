package middlewares

import (
	"github.com/zatrano/framework/configs/sessionconfig"

	"github.com/gofiber/fiber/v3"
)

// v3: işleyici imzası func(fiber.Ctx) error
func GuestMiddleware(c fiber.Ctx) error {
	sess, err := sessionconfig.SessionStart(c)
	if err != nil {
		return c.Next()
	}

	userIDRaw := sess.Get("user_id")
	if userIDRaw == nil {
		return c.Next()
	}

	userTypeRaw := sess.Get("user_type_id")
	var userTypeID uint
	switch v := userTypeRaw.(type) {
	case uint:
		userTypeID = v
	case int:
		userTypeID = uint(v)
	case int64:
		userTypeID = uint(v)
	default:
		return c.Redirect().Status(fiber.StatusSeeOther).To("/panel/anasayfa")
	}

	if userTypeID == 1 {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/home")
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To("/panel/anasayfa")
}
