package handlers

import (
	"github.com/zatrano/framework/packages/apierrors"
	"github.com/zatrano/framework/packages/i18n"
	"github.com/zatrano/framework/requests"
	"github.com/zatrano/framework/services"

	"github.com/gofiber/fiber/v3"
)

// UserAPIHandler — authenticated user işlemleri (/api/v1/user/*)
type UserAPIHandler struct {
	authService services.IAuthService
}

func NewUserAPIHandler(auth services.IAuthService) *UserAPIHandler {
	return &UserAPIHandler{authService: auth}
}

// Profile — kendi profil bilgilerini döner
func (h *UserAPIHandler) Profile(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok || userID == 0 {
		return apierrors.Send(c, apierrors.Unauthorized(i18n.T(c, "auth.session_expired")))
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
		"provider":       user.Provider,
		"created_at":     user.CreatedAt,
	})
}

// UpdateProfile — profil bilgilerini günceller
func (h *UserAPIHandler) UpdateProfile(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok || userID == 0 {
		return apierrors.Send(c, apierrors.Unauthorized(i18n.T(c, "auth.session_expired")))
	}
	var req requests.UpdateInfoRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierrors.Send(c, apierrors.BadRequest(i18n.T(c, "error.bad_request")))
	}
	if err := h.authService.UpdateUserInfo(c.Context(), userID, req.Name, req.Email); err != nil {
		if err == services.ErrEmailAlreadyExists {
			return apierrors.Send(c, apierrors.Conflict(i18n.T(c, "auth.email_exists")))
		}
		return apierrors.Send(c, apierrors.Internal(i18n.T(c, "crud.error")))
	}
	return apierrors.OK(c, fiber.Map{"message": i18n.T(c, "auth.profile_updated")})
}

// ChangePassword — şifre değişikliği
func (h *UserAPIHandler) ChangePassword(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok || userID == 0 {
		return apierrors.Send(c, apierrors.Unauthorized(i18n.T(c, "auth.session_expired")))
	}
	var req requests.UpdatePasswordRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierrors.Send(c, apierrors.BadRequest(i18n.T(c, "error.bad_request")))
	}
	if err := h.authService.UpdatePassword(c.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		switch err {
		case services.ErrCurrentPasswordIncorrect:
			return apierrors.Send(c, apierrors.BadRequest("Mevcut şifre hatalı"))
		case services.ErrPasswordSameAsOld:
			return apierrors.Send(c, apierrors.BadRequest("Yeni şifre eskiyle aynı olamaz"))
		default:
			return apierrors.Send(c, apierrors.Internal(i18n.T(c, "crud.error")))
		}
	}
	return apierrors.OK(c, fiber.Map{"message": i18n.T(c, "auth.password_updated")})
}
