package middlewares

import (
	"github.com/zatrano/framework/configs/sessionconfig"
	"github.com/zatrano/framework/packages/flashmessages"

	"github.com/gofiber/fiber/v3"
)

func UserTypeMiddleware(allowedTypes ...uint) fiber.Handler {
	return func(c fiber.Ctx) error {
		val := c.Locals("authUser")
		if val == nil {
			_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Oturum bulunamadı")
			return c.Redirect().Status(fiber.StatusSeeOther).To("/auth/login")
		}

		user, ok := val.(AuthUser)
		if !ok {
			_ = sessionconfig.DestroySession(c)
			_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Oturum bilgileri geçersiz")
			return c.Redirect().Status(fiber.StatusSeeOther).To("/auth/login")
		}

		allowed := false
		for _, t := range allowedTypes {
			if user.UserTypeID == t {
				allowed = true
				break
			}
		}

		if !allowed {
			_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
				"Bu sayfaya erişim yetkiniz bulunmamaktadır.")
			if user.UserTypeID == 1 {
				return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/home")
			}
			return c.Redirect().Status(fiber.StatusSeeOther).To("/panel/anasayfa")
		}

		return c.Next()
	}
}
