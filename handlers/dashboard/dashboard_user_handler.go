package handlers

import (
	"strconv"
	"net/http"
	"strings"

	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/packages/flashmessages"
	"github.com/zatrano/framework/packages/formflash"
	"github.com/zatrano/framework/packages/queryparams"
	"github.com/zatrano/framework/packages/renderer"
	"github.com/zatrano/framework/requests"
	"github.com/zatrano/framework/services"

	"github.com/gofiber/fiber/v3"
)

type DashboardUserHandler struct {
	userService     services.IUserService
	userTypeService services.IUserTypeService
}

func NewDashboardUserHandler(userService services.IUserService, userTypeService services.IUserTypeService) *DashboardUserHandler {
	return &DashboardUserHandler{
		userService:     userService,
		userTypeService: userTypeService,
	}
}

func (h *DashboardUserHandler) ListUsers(c fiber.Ctx) error {
	params, fieldErrors, err := requests.ParseAndValidateUserList(c)
	// Ortak Params map'i (pagination + filtreler için)
	var userTypeID uint
	if params.UserTypeID != nil {
		userTypeID = *params.UserTypeID
	}
	paramsMap := fiber.Map{
		"Name":          params.Name,
		"Email":         params.Email,
		"IsActive":      params.IsActive,
		"UserTypeId":    userTypeID,
		"SortBy":        params.SortBy,
        "OrderBy":       params.OrderBy,
		"Page":          params.Page,
		"PerPage":       params.PerPage,
		"EmailVerified": params.EmailVerified,
	}
	if err != nil {
		renderData := fiber.Map{
			"Title":            "Kullanıcılar",
			"ValidationErrors": fieldErrors,
			"Params":           paramsMap,
			"Result": &queryparams.PaginatedResult{
				Data: []models.User{},
				Meta: queryparams.PaginationMeta{
					CurrentPage: params.Page,
					PerPage:     params.PerPage,
					TotalItems:  0,
					TotalPages:  0,
				},
			},
		}
		return renderer.Render(c, "dashboard/users/list", "layouts/app", renderData, http.StatusBadRequest)
	}

	paginatedResult, err := h.userService.GetAllUsers(c.Context(), params)

	// Kullanıcı tiplerini filtreleme dropdown'ı için getir
	userTypesResult, _ := h.userTypeService.GetAllUserTypes(c.Context(), requests.UserTypeListParams{
		Page:    1,
		PerPage: 1000,
		SortBy:  "name",
		OrderBy: "asc",
	})

	userTypes := []models.UserType{}
	if userTypesResult != nil {
		if data, ok := userTypesResult.Data.([]models.UserType); ok {
			userTypes = data
		}
	}

	renderData := fiber.Map{
		"Title":     "Kullanıcılar",
		"Result":    paginatedResult,
		"UserTypes": userTypes,
		"Params":    paramsMap,
	}

	if err != nil {
		renderData[renderer.FlashErrorKeyView] = "Kullanıcılar getirilirken bir hata oluştu."
		renderData["Result"] = &queryparams.PaginatedResult{
			Data: []models.User{},
			Meta: queryparams.PaginationMeta{
				CurrentPage: params.Page,
				PerPage:     params.PerPage,
				TotalItems:  0,
				TotalPages:  0,
			},
		}
	}

	return renderer.Render(c, "dashboard/users/list", "layouts/app", renderData, http.StatusOK)
}

func (h *DashboardUserHandler) ShowCreateUser(c fiber.Ctx) error {
	userTypesResult, _ := h.userTypeService.GetAllUserTypes(c.Context(), requests.UserTypeListParams{
		Page:    1,
		PerPage: 1000,
		SortBy:  "name",
		OrderBy: "asc",
	})

	userTypes := []models.UserType{}
	if userTypesResult != nil {
		if data, ok := userTypesResult.Data.([]models.UserType); ok {
			userTypes = data
		}
	}

	return renderer.Render(c, "dashboard/users/create", "layouts/app", fiber.Map{
		"Title":     "Yeni Kullanıcı Ekle",
		"UserTypes": userTypes,
	})
}

func (h *DashboardUserHandler) CreateUser(c fiber.Ctx) error {
	// Form verilerini map olarak al
	formData := make(map[string]string)
	args := c.Request().PostArgs()
	args.VisitAll(func(key, value []byte) {
		formData[string(key)] = string(value)
	})

	// Create için özel validation
	req, fieldErrors, err := requests.ParseAndValidateCreateUserRequest(c)
	if err != nil {
		// Form verilerini kaydet
		formflash.SetData(c, formData)

		// Field-specific hataları kaydet
		formflash.SetValidationErrors(c, fieldErrors)

		// Genel hata mesajı
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())

		return c.Redirect().To("/dashboard/users/create")
	}

	// YENİ: CreateFromRequest kullan (CreateUserRequest tipinde)
	if err := h.userService.CreateUser(c.Context(), req); err != nil {
		// Servis hatası - form verilerini koru
		formflash.SetData(c, formData)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Kullanıcı oluşturulamadı: "+err.Error())
		return c.Redirect().To("/dashboard/users/create")
	}

	// BAŞARILI - form verilerini temizle
	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Kullanıcı başarıyla oluşturuldu.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/users")
}

func (h *DashboardUserHandler) ShowUpdateUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString("Geçersiz kullanıcı ID")
	}

	user, err := h.userService.GetUserByID(c.Context(), uint(id))
	if err != nil {
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Kullanıcı bulunamadı.")
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/users")
	}

	userTypesResult, _ := h.userTypeService.GetAllUserTypes(c.Context(), requests.UserTypeListParams{
		Page:    1,
		PerPage: 1000,
		SortBy:  "name",
		OrderBy: "asc",
	})

	userTypes := []models.UserType{}
	if userTypesResult != nil {
		if data, ok := userTypesResult.Data.([]models.UserType); ok {
			userTypes = data
		}
	}

	return renderer.Render(c, "dashboard/users/update", "layouts/app", fiber.Map{
		"Title":     "Kullanıcı Düzenle",
		"User":      user,
		"UserTypes": userTypes,
	})
}

func (h *DashboardUserHandler) UpdateUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz kullanıcı ID")
	}

	// Form verilerini map olarak al
	formData := make(map[string]string)
	args := c.Request().PostArgs()
	args.VisitAll(func(key, value []byte) {
		formData[string(key)] = string(value)
	})

	// Update için özel validation
	req, fieldErrors, err := requests.ParseAndValidateUpdateUserRequest(c)
	if err != nil {
		// Form verilerini kaydet
		formflash.SetData(c, formData)

		// Field-specific hataları kaydet
		formflash.SetValidationErrors(c, fieldErrors)

		// Genel hata mesajı
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, err.Error())

		return c.Redirect().To("/dashboard/users/update/" + c.Params("id"))
	}

	// YENİ: UpdateFromRequest kullan (UpdateUserRequest tipinde)
	if err := h.userService.UpdateUser(c.Context(), uint(id), req); err != nil {
		// Servis hatası - form verilerini koru
		formflash.SetData(c, formData)
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, "Kullanıcı güncellenemedi: "+err.Error())
		return c.Redirect().To("/dashboard/users/update/" + c.Params("id"))
	}

	// BAŞARILI - form verilerini temizle
	formflash.ClearData(c)
	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Kullanıcı başarıyla güncellendi.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/users")
}

func (h *DashboardUserHandler) DeleteUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Geçersiz kullanıcı ID")
	}

	if err := h.userService.DeleteUser(c.Context(), uint(id)); err != nil {
		errMsg := "Kullanıcı silinemedi: " + err.Error()
		if strings.Contains(c.Get("Accept"), "application/json") {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": errMsg})
		}
		flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey, errMsg)
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard/users")
	}

	if strings.Contains(c.Get("Accept"), "application/json") {
		return c.JSON(fiber.Map{"message": "Kullanıcı başarıyla silindi."})
	}

	flashmessages.SetFlashMessage(c, flashmessages.FlashSuccessKey, "Kullanıcı başarıyla silindi.")
	return c.Redirect().Status(fiber.StatusFound).To("/dashboard/users")
}
