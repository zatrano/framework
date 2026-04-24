package handlers

import (
	"strconv"

	"github.com/zatrano/framework/packages/apierrors"
	"github.com/zatrano/framework/packages/i18n"
	"github.com/zatrano/framework/requests"
	"github.com/zatrano/framework/services"

	"github.com/gofiber/fiber/v3"
)

// AdminUserAPIHandler — admin kullanıcı yönetimi (/api/v1/admin/users/*)
type AdminUserAPIHandler struct {
	userService services.IUserService
}

func NewAdminUserAPIHandler(user services.IUserService) *AdminUserAPIHandler {
	return &AdminUserAPIHandler{userService: user}
}

// ListUsers — tüm kullanıcıları listeler (paginated)
func (h *AdminUserAPIHandler) ListUsers(c fiber.Ctx) error {
	params, _, _ := requests.ParseAndValidateUserList(c)
	result, err := h.userService.GetAllUsers(c.Context(), params)
	if err != nil {
		return apierrors.Send(c, apierrors.Internal(i18n.T(c, "crud.error")))
	}
	return apierrors.OK(c, result.Data, result.Meta)
}

// GetUser — tek kullanıcı detayı
func (h *AdminUserAPIHandler) GetUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return apierrors.Send(c, apierrors.BadRequest("Geçersiz kullanıcı ID"))
	}
	user, err := h.userService.GetUserByID(c.Context(), uint(id))
	if err != nil {
		return apierrors.Send(c, apierrors.NotFound("Kullanıcı"))
	}
	return apierrors.OK(c, user)
}

// CreateUser — yeni kullanıcı oluşturur
func (h *AdminUserAPIHandler) CreateUser(c fiber.Ctx) error {
	var req requests.CreateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierrors.Send(c, apierrors.BadRequest(i18n.T(c, "error.bad_request")))
	}
	if err := h.userService.CreateUser(c.Context(), req); err != nil {
		return apierrors.Send(c, apierrors.Internal(i18n.T(c, "crud.error")))
	}
	return apierrors.Created(c, fiber.Map{"message": i18n.T(c, "crud.created")})
}

// UpdateUser — kullanıcı günceller
func (h *AdminUserAPIHandler) UpdateUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return apierrors.Send(c, apierrors.BadRequest("Geçersiz kullanıcı ID"))
	}
	var req requests.UpdateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierrors.Send(c, apierrors.BadRequest(i18n.T(c, "error.bad_request")))
	}
	if err := h.userService.UpdateUser(c.Context(), uint(id), req); err != nil {
		return apierrors.Send(c, apierrors.Internal(i18n.T(c, "crud.error")))
	}
	return apierrors.OK(c, fiber.Map{"message": i18n.T(c, "crud.updated")})
}

// DeleteUser — kullanıcı siler
func (h *AdminUserAPIHandler) DeleteUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return apierrors.Send(c, apierrors.BadRequest("Geçersiz kullanıcı ID"))
	}
	if err := h.userService.DeleteUser(c.Context(), uint(id)); err != nil {
		return apierrors.Send(c, apierrors.Internal(i18n.T(c, "crud.error")))
	}
	return apierrors.NoContent(c)
}
