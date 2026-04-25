package handlers

import (
	"net/http"
	"strings"

	"github.com/zatrano/framework/packages/apierrors"
	"github.com/zatrano/framework/services"

	"github.com/gofiber/fiber/v3"
)

type AuthAPIHandler struct {
	authService services.IAuthService
	jwtService  services.IJWTService
}

func NewAuthAPIHandler(auth services.IAuthService, jwt services.IJWTService) *AuthAPIHandler {
	return &AuthAPIHandler{authService: auth, jwtService: jwt}
}

// Login için godoc
// @Summary     Kullanıcı girişi (JWT)
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body requests.LoginRequest true "Giriş bilgileri"
// @Success     200 {object} map[string]interface{}
// @Failure     400,401,422 {object} map[string]interface{}
// @Router      /api/v1/auth/login [post]
func (h *AuthAPIHandler) Login(c fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return apierrors.Send(c, apierrors.BadRequest("Geçersiz JSON formatı"))
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Password = strings.TrimSpace(req.Password)
	if req.Email == "" || req.Password == "" {
		return apierrors.Send(c, apierrors.BadRequest("email ve password alanları zorunlu"))
	}

	user, err := h.authService.Authenticate(req.Email, req.Password)
	if err != nil {
		switch err {
		case services.ErrInvalidCredentials:
			return apierrors.Send(c, apierrors.Unauthorized("Kullanıcı adı veya şifre hatalı"))
		case services.ErrUserInactive:
			return apierrors.Send(c, apierrors.Forbidden("Hesabınız aktif değil"))
		default:
			return apierrors.Send(c, apierrors.Unauthorized("Giriş başarısız"))
		}
	}

	accessToken, err := h.jwtService.GenerateToken(user)
	if err != nil {
		return apierrors.Send(c, apierrors.Internal("Token oluşturulamadı"))
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken(user)
	if err != nil {
		return apierrors.Send(c, apierrors.Internal("Refresh token oluşturulamadı"))
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "Bearer",
			"user": fiber.Map{
				"id":           user.ID,
				"name":         user.Name,
				"email":        user.Email,
				"user_type_id": user.UserTypeID,
			},
		},
	})
}

// Register için godoc
// @Summary     Kullanıcı kaydı
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body map[string]string true "name,email,password,confirm_password"
// @Success     201 {object} map[string]interface{}
// @Failure     400,409,422 {object} map[string]interface{}
// @Router      /api/v1/auth/register [post]
func (h *AuthAPIHandler) Register(c fiber.Ctx) error {
	var req struct {
		Name            string `json:"name"`
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return apierrors.Send(c, apierrors.BadRequest("Geçersiz JSON formatı"))
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Name == "" || req.Email == "" || req.Password == "" || req.ConfirmPassword == "" {
		return apierrors.Send(c, apierrors.BadRequest("name, email, password ve confirm_password alanları zorunlu"))
	}
	if len(req.Name) < 3 {
		return apierrors.Send(c, apierrors.BadRequest("name en az 3 karakter olmalıdır"))
	}
	if len(req.Password) < 8 {
		return apierrors.Send(c, apierrors.BadRequest("password en az 8 karakter olmalıdır"))
	}
	if req.Password != req.ConfirmPassword {
		return apierrors.Send(c, apierrors.BadRequest("password ve confirm_password eşleşmiyor"))
	}

	err := h.authService.RegisterUser(c.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		switch err {
		case services.ErrEmailAlreadyExists:
			return apierrors.Send(c, apierrors.Conflict("Bu e-posta adresi zaten kayıtlı"))
		default:
			return apierrors.Send(c, apierrors.UnprocessableEntity("Kayıt işlemi başarısız", nil))
		}
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"data": fiber.Map{
			"message": "Kayıt başarılı. Lütfen e-posta adresinizi doğrulayın.",
			"email":   req.Email,
		},
	})
}

// Refresh için godoc
// @Summary Refresh token ile yeni erişim tokenı al
// @Tags    auth
// @Accept  json
// @Produce json
// @Param   body body map[string]string true "refresh_token"
// @Success 200 {object} map[string]interface{}
// @Router  /api/v1/auth/refresh [post]
func (h *AuthAPIHandler) Refresh(c fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.RefreshToken == "" {
		return apierrors.Send(c, apierrors.BadRequest("refresh_token alanı zorunlu"))
	}

	newToken, err := h.jwtService.RefreshAccessToken(body.RefreshToken)
	if err != nil {
		return apierrors.Send(c, apierrors.Unauthorized(err.Error()))
	}

	return apierrors.OK(c, fiber.Map{
		"access_token": newToken,
		"token_type":   "Bearer",
	})
}

// VerifyEmail için godoc
// @Summary E-posta doğrulama
// @Tags    auth
// @Accept  json
// @Produce json
// @Param   body body map[string]string true "token"
// @Success 200 {object} map[string]interface{}
// @Router  /api/v1/auth/verify-email [post]
func (h *AuthAPIHandler) VerifyEmail(c fiber.Ctx) error {
	var body struct {
		Token string `json:"token"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return apierrors.Send(c, apierrors.BadRequest("Geçersiz JSON formatı"))
	}
	body.Token = strings.TrimSpace(body.Token)
	if body.Token == "" {
		return apierrors.Send(c, apierrors.BadRequest("token alanı zorunlu"))
	}

	if err := h.authService.VerifyEmail(body.Token); err != nil {
		switch err {
		case services.ErrTokenExpired:
			return apierrors.Send(c, apierrors.UnprocessableEntity("Token süresi dolmuş", nil))
		case services.ErrUserNotFound:
			return apierrors.Send(c, apierrors.NotFound("Kullanıcı"))
		default:
			return apierrors.Send(c, apierrors.UnprocessableEntity("E-posta doğrulama başarısız", nil))
		}
	}

	return apierrors.OK(c, fiber.Map{
		"message": "E-posta başarıyla doğrulandı",
	})
}

// ResendVerification için godoc
// @Summary Doğrulama e-postasını yeniden gönder
// @Tags    auth
// @Accept  json
// @Produce json
// @Param   body body map[string]string true "email"
// @Success 200 {object} map[string]interface{}
// @Router  /api/v1/auth/resend-verification [post]
func (h *AuthAPIHandler) ResendVerification(c fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return apierrors.Send(c, apierrors.BadRequest("Geçersiz JSON formatı"))
	}
	body.Email = strings.ToLower(strings.TrimSpace(body.Email))
	if body.Email == "" {
		return apierrors.Send(c, apierrors.BadRequest("email alanı zorunlu"))
	}

	if err := h.authService.ResendVerificationLink(body.Email); err != nil {
		switch err {
		case services.ErrUserNotFound:
			return apierrors.Send(c, apierrors.NotFound("Kullanıcı"))
		default:
			return apierrors.Send(c, apierrors.UnprocessableEntity("Doğrulama e-postası gönderilemedi", nil))
		}
	}

	return apierrors.OK(c, fiber.Map{
		"message": "Doğrulama e-postası gönderildi",
	})
}

// ForgotPassword için godoc
// @Summary Şifre sıfırlama bağlantısı gönder
// @Tags    auth
// @Accept  json
// @Produce json
// @Param   body body map[string]string true "email"
// @Success 200 {object} map[string]interface{}
// @Router  /api/v1/auth/forgot-password [post]
func (h *AuthAPIHandler) ForgotPassword(c fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return apierrors.Send(c, apierrors.BadRequest("Geçersiz JSON formatı"))
	}
	body.Email = strings.ToLower(strings.TrimSpace(body.Email))
	if body.Email == "" {
		return apierrors.Send(c, apierrors.BadRequest("email alanı zorunlu"))
	}

	if err := h.authService.SendPasswordResetLink(body.Email); err != nil {
		switch err {
		case services.ErrUserNotFound:
			return apierrors.Send(c, apierrors.NotFound("Kullanıcı"))
		default:
			return apierrors.Send(c, apierrors.UnprocessableEntity("Şifre sıfırlama bağlantısı gönderilemedi", nil))
		}
	}

	return apierrors.OK(c, fiber.Map{
		"message": "Şifre sıfırlama bağlantısı gönderildi",
	})
}

// ResetPassword için godoc
// @Summary Token ile şifre sıfırla
// @Tags    auth
// @Accept  json
// @Produce json
// @Param   body body map[string]string true "token,new_password,confirm_password"
// @Success 200 {object} map[string]interface{}
// @Router  /api/v1/auth/reset-password [post]
func (h *AuthAPIHandler) ResetPassword(c fiber.Ctx) error {
	var body struct {
		Token           string `json:"token"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return apierrors.Send(c, apierrors.BadRequest("Geçersiz JSON formatı"))
	}
	body.Token = strings.TrimSpace(body.Token)
	if body.Token == "" || body.NewPassword == "" || body.ConfirmPassword == "" {
		return apierrors.Send(c, apierrors.BadRequest("token, new_password ve confirm_password alanları zorunlu"))
	}
	if len(body.NewPassword) < 8 {
		return apierrors.Send(c, apierrors.BadRequest("new_password en az 8 karakter olmalıdır"))
	}
	if body.NewPassword != body.ConfirmPassword {
		return apierrors.Send(c, apierrors.BadRequest("new_password ve confirm_password eşleşmiyor"))
	}

	if err := h.authService.ResetPassword(body.Token, body.NewPassword); err != nil {
		switch err {
		case services.ErrTokenExpired:
			return apierrors.Send(c, apierrors.UnprocessableEntity("Token süresi dolmuş", nil))
		case services.ErrUserNotFound:
			return apierrors.Send(c, apierrors.NotFound("Kullanıcı"))
		default:
			return apierrors.Send(c, apierrors.UnprocessableEntity("Şifre sıfırlama başarısız", nil))
		}
	}

	return apierrors.OK(c, fiber.Map{
		"message": "Şifre başarıyla sıfırlandı",
	})
}

// Me için godoc
// @Summary  Oturum açmış kullanıcının bilgileri
// @Tags     auth
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} map[string]interface{}
// @Router   /api/v1/auth/me [get]
func (h *AuthAPIHandler) Me(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok || userID == 0 {
		return apierrors.Send(c, apierrors.Unauthorized("Oturum bilgisi bulunamadı"))
	}
	user, err := h.authService.GetUserProfile(c.Context(), userID)
	if err != nil {
		return apierrors.Send(c, apierrors.NotFound("Kullanıcı"))
	}
	return apierrors.OK(c, fiber.Map{
		"id":             user.ID,
		"name":           user.Name,
		"email":          user.Email,
		"user_type_id":   user.UserTypeID,
		"email_verified": user.EmailVerified,
		"is_active":      user.IsActive,
		"created_at":     user.CreatedAt,
	})
}

// Logout için godoc
// @Summary JWT istemci tarafı çıkış
// @Tags    auth
// @Security BearerAuth
// @Accept  json
// @Produce json
// @Param   body body map[string]string false "refresh_token (opsiyonel)"
// @Success 200 {object} map[string]interface{}
// @Router  /api/v1/auth/logout [post]
func (h *AuthAPIHandler) Logout(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && strings.TrimSpace(parts[1]) != "" {
		_ = h.jwtService.RevokeToken(strings.TrimSpace(parts[1]))
	}

	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.Bind().JSON(&body)
	body.RefreshToken = strings.TrimSpace(body.RefreshToken)
	if body.RefreshToken != "" {
		_ = h.jwtService.RevokeToken(body.RefreshToken)
	}

	return apierrors.OK(c, fiber.Map{
		"message": "Çıkış başarılı. Access ve varsa refresh token iptal edildi.",
	})
}
