package handlers

import (
	"net/http"

	"github.com/zatrano/framework/packages/apierrors"
	"github.com/zatrano/framework/requests"
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
	var req requests.LoginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierrors.Send(c, apierrors.BadRequest("Geçersiz JSON formatı"))
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
