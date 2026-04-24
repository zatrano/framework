package currentuser

import (
	"context"
	"reflect"

	"github.com/gofiber/fiber/v3"
)

type contextKey string

const (
	ContextUserIDKey     contextKey = "user_id"
	ContextUserEmailKey  contextKey = "user_email"
	ContextUserTypeIDKey contextKey = "user_type_id"
)

type CurrentUser struct {
	ID         uint
	Email      string
	UserTypeID uint
}

// v3: FromFiber — fiber.Ctx artık context.Context'i implement ediyor.
// currentUser, authUser ve diğer tipler Locals'tan alınır.
// SetUserContext/UserContext kaldırıldı; bunun yerine Locals kullanılır.
func FromFiber(c fiber.Ctx) CurrentUser {
	// 1. "currentUser" Locals anahtarını kontrol et (AuthMiddleware tarafından set edilir)
	if cu, ok := c.Locals("currentUser").(CurrentUser); ok {
		return cu
	}

	// 2. "authUser" Locals anahtarını yansımayla kontrol et
	authUserVal := c.Locals("authUser")
	if authUserVal == nil {
		return CurrentUser{}
	}

	if au, ok := authUserVal.(CurrentUser); ok {
		return au
	}
	if au, ok := authUserVal.(fiber.Map); ok {
		return CurrentUser{
			ID:         convertToUint(au["ID"]),
			Email:      convertToString(au["Email"]),
			UserTypeID: convertToUint(au["UserTypeID"]),
		}
	}
	if au, ok := authUserVal.(map[string]interface{}); ok {
		return CurrentUser{
			ID:         convertToUint(au["ID"]),
			Email:      convertToString(au["Email"]),
			UserTypeID: convertToUint(au["UserTypeID"]),
		}
	}

	// Reflection yedeği (AuthUser struct vb.)
	rv := reflect.ValueOf(authUserVal)
	if rv.Kind() == reflect.Struct {
		idField := rv.FieldByName("ID")
		emailField := rv.FieldByName("Email")
		userTypeIDField := rv.FieldByName("UserTypeID")
		if idField.IsValid() {
			var id, userTypeID uint
			var email string
			switch idField.Kind() {
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				id = uint(idField.Uint())
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				id = uint(idField.Int())
			}
			if emailField.IsValid() && emailField.Kind() == reflect.String {
				email = emailField.String()
			}
			if userTypeIDField.IsValid() {
				switch userTypeIDField.Kind() {
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					userTypeID = uint(userTypeIDField.Uint())
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					userTypeID = uint(userTypeIDField.Int())
				}
			}
			return CurrentUser{ID: id, Email: email, UserTypeID: userTypeID}
		}
	}
	return CurrentUser{}
}

// v3: fiber.Ctx context.Context implement ettiği için context.WithValue yerine
// Locals kullanılmalıdır. Bu fonksiyon geriye dönük uyumluluk içindir.
func SetToContext(ctx context.Context, user CurrentUser) context.Context {
	ctx = context.WithValue(ctx, ContextUserIDKey, user.ID)
	ctx = context.WithValue(ctx, ContextUserEmailKey, user.Email)
	ctx = context.WithValue(ctx, ContextUserTypeIDKey, user.UserTypeID)
	return ctx
}

func FromContext(ctx context.Context) CurrentUser {
	var cu CurrentUser
	if v := ctx.Value(ContextUserIDKey); v != nil {
		cu.ID = convertToUint(v)
	}
	if v := ctx.Value(ContextUserEmailKey); v != nil {
		if s, ok := v.(string); ok {
			cu.Email = s
		}
	}
	if v := ctx.Value(ContextUserTypeIDKey); v != nil {
		cu.UserTypeID = convertToUint(v)
	}
	return cu
}

func convertToUint(val interface{}) uint {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case uint:
		return v
	case int:
		return uint(v)
	case float64:
		return uint(v)
	case float32:
		return uint(v)
	case int64:
		return uint(v)
	case uint64:
		return uint(v)
	}
	return 0
}

func convertToString(val interface{}) string {
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}
