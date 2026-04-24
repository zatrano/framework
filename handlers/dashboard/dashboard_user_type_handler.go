package handlers

import (
	"strconv"
	"net/http"
	"strings"

	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/packages/flashmessages"
	"github.com/zatrano/framework/packages/formflash"
	"github.com/zatrano/framework/packages/renderer"
	"github.com/zatrano/framework/requests"
	"github.com/zatrano/framework/services"

	"github.com/gofiber/fiber/v3"
)

type DashboardUserTypeHandler struct {
	userTypeService services.IUserTypeService
}

func NewDashboardUserTypeHandler(userTypeService services.IUserTypeService) *DashboardUserTypeHandler {
	return &DashboardUserTypeHandler{
		userTypeService: userTypeService,
	}
}

func (h *DashboardUserTypeHandler) ListUserTypes(c fiber.Ctx) error {
	params, fieldErrors, err := requests.ParseAndValidateUserTypeList(c)
	if err != nil {
		renderData := fiber.Map{
			"Title":            "Kullanıcı Tipleri",
			"ValidationErrors": fieldErrors,
			"Params": fiber.Map{
				"Name":     params.Name,
				"IsActive": params.IsActive,
				"SortBy":   params.SortBy,
				"OrderBy":  params.OrderBy,
				"Page":     params.Page,
				"PerPage":  params.PerPage,
			},
			"Result": &requests.PaginatedResult{
				Data: []models.UserType{},
				Meta: requests.PaginationMeta{
					CurrentPage: params.Page,
					PerPage:     params.PerPage,
					TotalItems:  0,
					TotalPages:  0,
				},
			},
		}
		return renderer.Render(c, "dashboard/user-types/list", "layouts/app", renderData, http.StatusBadRequest)
	}

	paginatedResult, err := h.userTypeService.GetAllUserTypes(c.Context(), params)

	renderData := fiber.Map{
		"Title":  "Kullanıcı Tipleri",
		"Result": paginatedResult,
		"Params": fiber.Map{
			"name":      params.Name,
			"is_active": params.IsActive,
			"sortBy":    params.SortBy,
			"orderBy":   params.OrderBy,
			"page":      params.Page,
			"perPage":   params.PerPage,
		},
	}

	if err != nil {
		renderData[renderer.FlashErrorKeyView] = "Kullanıcı Tipleri getirilirken bir hata oluştu."
		renderData["Result"] = &requests.PaginatedResult{
			Data: []models.UserType{},
			Meta: requests.PaginationMeta{
				CurrentPage: params.Page,
				PerPage:     params.PerPage,
				TotalItems:  0,
				TotalPages:  0,
			},
		}
	}

	return renderer.Render(c, "dashboard/user-types/list", "layouts/app", renderData, http.StatusOK)
}

func (h *DashboardUserTypeHandler) ShowCreateUserType(c fiber.Ctx) error {
	return renderer.Render(c, "dashboard/user-types/create", "layouts/app", fiber.Map{
		"Title": "Yeni Kullanıcı Tipi Ekle",
	})
}

func (h *DashboardUserTypeHandler) CreateUserType(c fiber.Ctx) error {
	formData := make(map[string]string)

	args := c.Request().PostArgs()
	args.VisitAll(func(key, value []byte) {
		formData[string(key)] = string(value)
	})

	req, fieldErrors, err := requests.ParseAndValidateUserTypeRequest(c)

	if err != nil {
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())

		return c.Redirect().To("/dashboard/user-types/create")
	}

	if err := h.userTypeService.CreateUserType(c.Context(), req); err != nil {
		formflash.SetData(c, formData)

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Kullanıcı Tipi oluşturulamadı: "+err.Error())

		return c.Redirect().To("/dashboard/user-types/create")
	}

	formflash.ClearData(c)

	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Kullanıcı Tipi başarıyla oluşturuldu.")

	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/user-types")
}

func (h *DashboardUserTypeHandler) ShowUpdateUserType(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString("Geçersiz Kullanıcı Tipi ID")
	}

	userType, err := h.userTypeService.GetUserTypeByID(c.Context(), uint(id))
	if err != nil {
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Kullanıcı Tipi bulunamadı.")

		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/user-types")
	}

	return renderer.Render(c, "dashboard/user-types/update", "layouts/app", fiber.Map{
		"Title":    "Kullanıcı Tipi Düzenle",
		"UserType": userType,
	})
}

func (h *DashboardUserTypeHandler) UpdateUserType(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz Kullanıcı Tipi ID")
	}

	formData := make(map[string]string)

	args := c.Request().PostArgs()
	args.VisitAll(func(key, value []byte) {
		formData[string(key)] = string(value)
	})

	req, fieldErrors, err := requests.ParseAndValidateUserTypeRequest(c)

	if err != nil {
		formflash.SetData(c, formData)
		formflash.SetValidationErrors(c, fieldErrors)

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())

		return c.Redirect().To("/dashboard/user-types/update/" + c.Params("id"))
	}

	if err := h.userTypeService.UpdateUserType(c.Context(), uint(id), req); err != nil {
		formflash.SetData(c, formData)

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Kullanıcı Tipi güncellenemedi: "+err.Error())

		return c.Redirect().To("/dashboard/user-types/update/" + c.Params("id"))
	}

	formflash.ClearData(c)

	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Kullanıcı Tipi başarıyla güncellendi.")

	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/user-types")
}

func (h *DashboardUserTypeHandler) DeleteUserType(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz Kullanıcı Tipi ID")
	}

	if err := h.userTypeService.DeleteUserType(c.Context(), uint(id)); err != nil {
		errMsg := "Kullanıcı Tipi silinemedi: " + err.Error()

		if strings.Contains(c.Get("Accept"), "application/json") {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": errMsg})
		}

		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, errMsg)

		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/user-types")
	}

	if strings.Contains(c.Get("Accept"), "application/json") {
		return c.JSON(fiber.Map{"message": "Kullanıcı Tipi başarıyla silindi."})
	}

	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Kullanıcı Tipi başarıyla silindi.")

	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/user-types")
}
