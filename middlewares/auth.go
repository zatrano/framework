package middlewares

import (
	"strings"

	"github.com/zatrano/framework/configs/sessionconfig"
	"github.com/zatrano/framework/packages/currentuser"
	"github.com/zatrano/framework/packages/flashmessages"
	"github.com/zatrano/framework/services"

	"github.com/gofiber/fiber/v3"
)

type AuthUser struct {
	ID            uint
	Email         string
	Name          string
	UserTypeID    uint
	IsActive      bool
	EmailVerified bool
}

// v3: işleyici imzası func(fiber.Ctx) error

func AuthMiddleware(authService services.IAuthService) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, err := sessionconfig.GetUserIDFromSession(c)
		if err != nil || userID == 0 {
			_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
				"Oturum süresi dolmuş veya geçersiz.")
			return c.Redirect().Status(fiber.StatusSeeOther).To("/auth/login")
		}

		// v3: c artık context.Context — doğrudan geçilebilir
		user, err := authService.GetUserProfile(c.Context(), userID)
		if err != nil {
			_ = sessionconfig.DestroySession(c)
			_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
				"Kullanıcı bulunamadı, lütfen tekrar giriş yapın.")
			return c.Redirect().Status(fiber.StatusSeeOther).To("/auth/login")
		}

		if !user.EmailVerified {
			_ = sessionconfig.DestroySession(c)

			sess, _ := sessionconfig.SessionStart(c)
			sess.Set("pending_verification", true)
			sess.Set("user_email", user.Email)
			_ = sess.Save()

			_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
				"Lütfen e-posta adresinizi doğrulayınız. Doğrulama linki e-postanıza gönderilmiştir.")
			return c.Redirect().Status(fiber.StatusSeeOther).To("/auth/login")
		}

		if !user.IsActive {
			_ = sessionconfig.DestroySession(c)
			_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
				"Hesabınız pasif durumda. Lütfen yöneticinizle iletişime geçin.")
			return c.Redirect().Status(fiber.StatusSeeOther).To("/auth/login")
		}

		authUser := AuthUser{
			ID:            user.ID,
			Email:         user.Email,
			Name:          user.Name,
			UserTypeID:    user.UserTypeID,
			IsActive:      user.IsActive,
			EmailVerified: user.EmailVerified,
		}

		c.Locals("authUser", authUser)
		c.Locals("userID", user.ID)

		// v3: fiber.Ctx context.Context implement ediyor
		// currentuser bilgisini context'e set etmek için WithValue kullanıyoruz

		cu := currentuser.CurrentUser{
			ID:         user.ID,
			Email:      user.Email,
			Name:       user.Name,
			UserTypeID: user.UserTypeID,
		}
		c.Locals("currentUser", cu)

		if strings.HasPrefix(c.Path(), "/panel") && user.UserTypeID == 1 {
			return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard")
		}

		return c.Next()
	}
}
