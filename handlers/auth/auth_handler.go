package handlers

import (
	"net/http"

	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/configs/sessionconfig"
	"github.com/zatrano/framework/handlers/auth/oauth"
	"github.com/zatrano/framework/packages/flashmessages"
	"github.com/zatrano/framework/packages/formflash"
	"github.com/zatrano/framework/packages/renderer"
	"github.com/zatrano/framework/packages/turnstile"
	"github.com/zatrano/framework/requests"
	"github.com/zatrano/framework/services"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type AuthHandler struct {
	authService  services.IAuthService
	mailService  services.IMailService
	oauthHandler *oauth.OAuthHandler
}

func NewAuthHandler(authService services.IAuthService, mailService services.IMailService) *AuthHandler {
	factory := oauth.NewProviderFactory()
	oauthHandler := factory.CreateOAuthHandler(authService)

	return &AuthHandler{
		authService:  authService,
		mailService:  mailService,
		oauthHandler: oauthHandler,
	}
}

func (h *AuthHandler) getSessionUser(c fiber.Ctx) (uint, error) {
	if userID, ok := c.Locals("userID").(uint); ok {
		return userID, nil
	}
	sess, err := sessionconfig.SessionStart(c)
	if err != nil {
		return 0, err
	}
	switch v := sess.Get("user_id").(type) {
	case uint:
		return v, nil
	case int:
		return uint(v), nil
	case float64:
		return uint(v), nil
	default:
		return 0, fiber.ErrUnauthorized
	}
}

func (h *AuthHandler) destroySession(c fiber.Ctx) {
	sess, err := sessionconfig.SessionStart(c)
	if err != nil {
		logconfig.Log.Warn("Oturum yok edilemedi", zap.Error(err))
		return
	}
	_ = sess.Destroy()
}

func (h *AuthHandler) ShowLogin(c fiber.Ctx) error {
	sess, err := sessionconfig.SessionStart(c)

	var pendingVerification bool
	var userEmail string

	if err == nil {
		if notVerified := sess.Get("email_not_verified"); notVerified != nil {
			if b, ok := notVerified.(bool); ok && b {
				pendingVerification = true
				if email := sess.Get("user_email"); email != nil {
					userEmail = email.(string)
				}
			}
			sess.Delete("email_not_verified")
			sess.Delete("user_email")
		}

		_ = sess.Save()
	}

	return renderer.Render(c, "auth/login", "layouts/auth", fiber.Map{
		"Title":               "Giriş Yap",
		"PendingVerification": pendingVerification,
		"UserEmail":           userEmail,
	}, http.StatusOK)
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	req, fieldErrors, err := requests.ParseAndValidateLoginRequest(c)

	if err != nil {
		formData := map[string]string{
			"email": req.Email,
		}
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/auth/login")
	}

	user, err := h.authService.Authenticate(req.Email, req.Password)
	if err != nil {
		formData := map[string]string{
			"email": req.Email,
		}
		formflash.SetData(c, formData)

		var errorMsg string
		switch err {
		case services.ErrInvalidCredentials:
			errorMsg = "Kullanıcı adı veya şifre hatalı."
		case services.ErrUserInactive:
			errorMsg = "Hesabınız aktif değil."
		case services.ErrUserNotFound:
			errorMsg = "Kullanıcı bulunamadı."
		default:
			errorMsg = "Giriş yapılırken bir hata oluştu."
		}

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, errorMsg)
		return c.Redirect().To("/auth/login")
	}

	if !user.EmailVerified {
		sess, _ := sessionconfig.SessionStart(c)
		sess.Set("pending_verification", true)
		sess.Set("user_email", user.Email)
		_ = sess.Save()

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
			"Lütfen e-posta adresinizi doğrulayınız. Doğrulama linki e-postanıza gönderilmiştir.")
		return c.Redirect().To("/auth/login")
	}

	sess, err := sessionconfig.SessionStart(c)
	if err != nil {
		logconfig.Log.Error("Oturum başlatılamadı",
			zap.Uint("user_id", user.ID),
			zap.String("email", user.Email),
			zap.Error(err))

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Oturum başlatılamadı.")
		return c.Redirect().To("/auth/login")
	}

	sess.Set("user_id", user.ID)
	sess.Set("user_type_id", user.UserTypeID)
	sess.Set("is_active", user.IsActive)
	if err := sess.Save(); err != nil {
		logconfig.Log.Error("Oturum kaydedilemedi",
			zap.Uint("user_id", user.ID),
			zap.String("email", user.Email),
			zap.Error(err))

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Oturum kaydedilemedi.")
		return c.Redirect().To("/auth/login")
	}

	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Başarıyla giriş yapıldı")

	if user.UserTypeID == 1 {
		return c.Redirect().Status(fiber.StatusFound).To("/dashboard")
	}
	return c.Redirect().Status(fiber.StatusFound).To("/panel/anasayfa")
}

func (h *AuthHandler) ShowRegister(c fiber.Ctx) error {
	siteKey := envconfig.String("TURNSTILE_SITE_KEY", "")
	return renderer.Render(c, "auth/register", "layouts/auth", fiber.Map{
		"Title":            "Kayıt Ol",
		"TurnstileSiteKey": siteKey,
	}, http.StatusOK)
}

func (h *AuthHandler) Register(c fiber.Ctx) error {
	req, fieldErrors, err := requests.ParseAndValidateRegisterRequest(c)

	if err != nil {
		formData := map[string]string{
			"name":  req.Name,
			"email": req.Email,
		}
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/auth/register")
	}

	// Robot önlemi: Turnstile yapılandırılmışsa doğrula
	if siteKey := envconfig.String("TURNSTILE_SITE_KEY", ""); siteKey != "" {
		secret := envconfig.String("TURNSTILE_SECRET_KEY", "")
		if secret == "" {
			formflash.SetData(c, map[string]string{"name": req.Name, "email": req.Email})
			formflash.SetValidationErrors(c, map[string]string{"Turnstile": "Güvenlik doğrulaması yapılandırılmamış."})
			flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Kayıt şu an kullanılamıyor.")
			return c.Redirect().To("/auth/register")
		}
		if req.TurnstileResponse == "" {
			formData := map[string]string{"name": req.Name, "email": req.Email}
			formflash.SetData(c, formData)
			formflash.SetValidationErrors(c, map[string]string{"Turnstile": "Lütfen insan doğrulamasını tamamlayın."})
			flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Lütfen güvenlik doğrulamasını tamamlayın.")
			return c.Redirect().To("/auth/register")
		}
		resp, verifyErr := turnstile.Verify(secret, req.TurnstileResponse, c.IP())
		if verifyErr != nil || resp == nil || !resp.Success {
			formData := map[string]string{"name": req.Name, "email": req.Email}
			formflash.SetData(c, formData)
			formflash.SetValidationErrors(c, map[string]string{"Turnstile": "Doğrulama başarısız. Lütfen tekrar deneyin."})
			flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Güvenlik doğrulaması başarısız. Lütfen tekrar deneyin.")
			return c.Redirect().To("/auth/register")
		}
	}

	err = h.authService.RegisterUser(c.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		formData := map[string]string{
			"name":  req.Name,
			"email": req.Email,
		}
		formflash.SetData(c, formData)

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
			"Kayıt işlemi başarısız: "+err.Error())
		return c.Redirect().To("/auth/register")
	}

	formflash.ClearData(c)

	return renderer.Render(c, "auth/verify_email_notice", "layouts/auth", fiber.Map{
		"Title":     "Email Doğrulama",
		"UserEmail": req.Email,
		"Success":   "Kaydınız Başarıyla Oluşturuldu",
	}, http.StatusOK)
}

func (h *AuthHandler) Profile(c fiber.Ctx) error {
	userID, err := h.getSessionUser(c)
	if err != nil {
		h.destroySession(c)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
			"Geçersiz oturum, lütfen tekrar giriş yapın.")
		return c.Redirect().Status(fiber.StatusSeeOther).To("/auth/login")
	}

	user, err := h.authService.GetUserProfile(c.Context(), userID)
	if err != nil {
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
			"Profil bilgileri alınamadı.")
		return c.Redirect().To("/auth/profile")
	}

	return renderer.Render(c, "auth/profile", "layouts/auth", fiber.Map{
		"Title": "Profilim",
		"User":  user,
	}, http.StatusOK)
}

func (h *AuthHandler) UpdatePassword(c fiber.Ctx) error {
	userID, err := h.getSessionUser(c)
	if err != nil {
		h.destroySession(c)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
			"Geçersiz oturum bilgisi.")
		return c.Redirect().To("/auth/login")
	}

	req, fieldErrors, err := requests.ParseAndValidateUpdatePasswordRequest(c)

	if err != nil {
		formflash.SetValidationErrors(c, fieldErrors)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/auth/profile")
	}

	if err := h.authService.UpdatePassword(c.Context(), userID,
		req.CurrentPassword, req.NewPassword); err != nil {

		var errorMsg string
		switch err {
		case services.ErrCurrentPasswordIncorrect:
			errorMsg = "Mevcut şifreniz hatalı."
		case services.ErrPasswordTooShort:
			errorMsg = "Yeni şifre en az 8 karakter olmalıdır."
		case services.ErrPasswordSameAsOld:
			errorMsg = "Yeni şifre mevcut şifreden farklı olmalıdır."
		default:
			errorMsg = "Şifre güncellenirken bir hata oluştu."
		}

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, errorMsg)
		return c.Redirect().To("/auth/profile")
	}

	h.destroySession(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey,
		"Şifre başarıyla güncellendi. Lütfen yeni şifrenizle tekrar giriş yapın.")
	return c.Redirect().Status(fiber.StatusFound).To("/auth/login")
}

func (h *AuthHandler) ShowForgotPassword(c fiber.Ctx) error {
	return renderer.Render(c, "auth/forgot_password", "layouts/auth", fiber.Map{
		"Title": "Şifremi Unuttum",
	}, http.StatusOK)
}

func (h *AuthHandler) ForgotPassword(c fiber.Ctx) error {
	req, fieldErrors, err := requests.ParseAndValidateForgotPasswordRequest(c)

	if err != nil {
		formData := map[string]string{
			"email": req.Email,
		}
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/auth/forgot-password")
	}

	if err := h.authService.SendPasswordResetLink(req.Email); err != nil {
		formData := map[string]string{
			"email": req.Email,
		}
		formflash.SetData(c, formData)

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
			"Şifre sıfırlama bağlantısı gönderilemedi.")
		return c.Redirect().To("/auth/forgot-password")
	}

	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey,
		"Şifre sıfırlama bağlantısı gönderildi.")
	return c.Redirect().Status(fiber.StatusSeeOther).To("/auth/login")
}

func (h *AuthHandler) ShowResetPassword(c fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
			"Geçersiz veya eksik token.")
		return c.Redirect().To("/auth/forgot-password")
	}

	return renderer.Render(c, "auth/reset_password", "layouts/auth", fiber.Map{
		"Title": "Şifre Sıfırla",
		"Token": token,
	}, http.StatusOK)
}

func (h *AuthHandler) ResetPassword(c fiber.Ctx) error {
	req, fieldErrors, err := requests.ParseAndValidateResetPasswordRequest(c)

	if err != nil {
		formflash.SetValidationErrors(c, fieldErrors)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/auth/reset-password?token=" + req.Token)
	}

	if err := h.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
			"Şifre sıfırlama başarısız.")
		return c.Redirect().To("/auth/reset-password?token=" + req.Token)
	}

	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey,
		"Şifre sıfırlandı. Lütfen giriş yapın.")
	return c.Redirect().Status(fiber.StatusSeeOther).To("/auth/login")
}

func (h *AuthHandler) UpdateInfo(c fiber.Ctx) error {
	userID, err := h.getSessionUser(c)
	if err != nil {
		h.destroySession(c)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
			"Geçersiz oturum bilgisi.")
		return c.Redirect().To("/auth/login")
	}

	req, fieldErrors, err := requests.ParseAndValidateUpdateInfoRequest(c)

	if err != nil {
		formData := map[string]string{
			"name":  req.Name,
			"email": req.Email,
		}
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/auth/profile")
	}

	if err := h.authService.UpdateUserInfo(c.Context(), userID, req.Name, req.Email); err != nil {
		formData := map[string]string{
			"name":  req.Name,
			"email": req.Email,
		}
		formflash.SetData(c, formData)

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
			"Profil bilgileri güncellenirken bir hata oluştu.")
		return c.Redirect().To("/auth/profile")
	}

	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey,
		"Profil bilgileri güncellendi.")
	return c.Redirect().Status(fiber.StatusSeeOther).To("/auth/profile")
}

func (h *AuthHandler) VerifyEmail(c fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
			"Doğrulama tokeni eksik.")
		return c.Redirect().To("/auth/login")
	}

	if err := h.authService.VerifyEmail(token); err != nil {
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
			"Email doğrulama başarısız.")
		return c.Redirect().To("/auth/login")
	}

	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey,
		"Email başarıyla doğrulandı.")
	return c.Redirect().Status(fiber.StatusSeeOther).To("/auth/login")
}

func (h *AuthHandler) ShowResendVerification(c fiber.Ctx) error {
	email := c.Query("email")

	return renderer.Render(c, "auth/resend_verification", "layouts/auth", fiber.Map{
		"Title": "Email Doğrulama Linkini Yeniden Gönder",
		"Email": email,
	}, http.StatusOK)
}

func (h *AuthHandler) ResendVerification(c fiber.Ctx) error {
	req, fieldErrors, err := requests.ParseAndValidateResendVerificationRequest(c)

	if err != nil {
		formData := map[string]string{
			"email": req.Email,
		}
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())
		return c.Redirect().To("/auth/resend-verification")
	}

	if err := h.authService.ResendVerificationLink(req.Email); err != nil {
		formData := map[string]string{
			"email": req.Email,
		}
		formflash.SetData(c, formData)

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
			"Doğrulama linki gönderilemedi.")
		return c.Redirect().To("/auth/resend-verification")
	}

	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey,
		"Doğrulama linki e-posta adresinize gönderildi.")
	return c.Redirect().Status(fiber.StatusSeeOther).To("/auth/login")
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	h.destroySession(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey,
		"Başarıyla çıkış yapıldı.")
	return c.Redirect().Status(fiber.StatusFound).To("/auth/login")
}

func (h *AuthHandler) OAuthLogin(c fiber.Ctx) error {
	provider := c.Params("provider")
	return h.oauthHandler.HandleLogin(c, provider)
}

func (h *AuthHandler) OAuthCallback(c fiber.Ctx) error {
	provider := c.Params("provider")
	return h.oauthHandler.HandleCallback(c, provider)
}

func (h *AuthHandler) GoogleLogin(c fiber.Ctx) error {
	return h.oauthHandler.HandleLogin(c, "google")
}

func (h *AuthHandler) GoogleCallback(c fiber.Ctx) error {
	return h.oauthHandler.HandleCallback(c, "google")
}
