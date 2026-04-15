package middleware

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/framework/pkg/auth"
	"github.com/zatrano/framework/pkg/i18n"
)

// Locals keys used by authorization middleware.
const (
	LocalsUserID    = "zatrano.auth.user_id"
	LocalsUserRoles = "zatrano.auth.user_roles"
)

// ─── Permission-based middleware (RBAC) ─────────────────────────────────────

// Can returns middleware that checks whether the authenticated user has the
// given permission (via RBAC role→permission mapping). Responds with 403 JSON
// when denied, with an i18n-aware error message.
//
// Usage:
//
//	app.Get("/admin/users", middleware.Can(rbac, "users.view"), handler)
//	app.Post("/posts", middleware.Can(rbac, "posts.create"), handler)
func Can(rbac *auth.RBACManager, permission string) fiber.Handler {
	return func(c fiber.Ctx) error {
		roles, ok := c.Locals(LocalsUserRoles).([]string)
		if !ok || len(roles) == 0 {
			return forbiddenResponse(c, permission)
		}
		if !rbac.RolesHavePermission(roles, permission) {
			return forbiddenResponse(c, permission)
		}
		return c.Next()
	}
}

// CanAny returns middleware that passes if the user has ANY of the listed permissions.
//
// Usage:
//
//	app.Get("/reports", middleware.CanAny(rbac, "reports.view", "reports.export"), handler)
func CanAny(rbac *auth.RBACManager, permissions ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		roles, ok := c.Locals(LocalsUserRoles).([]string)
		if !ok || len(roles) == 0 {
			return forbiddenResponse(c, permissions[0])
		}
		if !rbac.RolesHaveAnyPermission(roles, permissions...) {
			return forbiddenResponse(c, permissions[0])
		}
		return c.Next()
	}
}

// CanAll returns middleware that passes only if the user has ALL of the listed permissions.
//
// Usage:
//
//	app.Delete("/admin/system", middleware.CanAll(rbac, "system.admin", "system.delete"), handler)
func CanAll(rbac *auth.RBACManager, permissions ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		roles, ok := c.Locals(LocalsUserRoles).([]string)
		if !ok || len(roles) == 0 {
			return forbiddenResponse(c, permissions[0])
		}
		if !rbac.RolesHaveAllPermissions(roles, permissions...) {
			return forbiddenResponse(c, permissions[0])
		}
		return c.Next()
	}
}

// HasRole returns middleware that checks whether the user has a specific role.
//
// Usage:
//
//	app.Get("/admin", middleware.HasRole("admin"), handler)
func HasRole(roleName string) fiber.Handler {
	return func(c fiber.Ctx) error {
		roles, ok := c.Locals(LocalsUserRoles).([]string)
		if !ok || len(roles) == 0 {
			return forbiddenResponse(c, "role:"+roleName)
		}
		for _, r := range roles {
			if r == roleName {
				return c.Next()
			}
		}
		return forbiddenResponse(c, "role:"+roleName)
	}
}

// HasAnyRole returns middleware that passes if the user has any of the listed roles.
//
// Usage:
//
//	app.Get("/dashboard", middleware.HasAnyRole("admin", "editor"), handler)
func HasAnyRole(roleNames ...string) fiber.Handler {
	set := make(map[string]bool, len(roleNames))
	for _, r := range roleNames {
		set[r] = true
	}
	return func(c fiber.Ctx) error {
		roles, ok := c.Locals(LocalsUserRoles).([]string)
		if !ok || len(roles) == 0 {
			return forbiddenResponse(c, "role")
		}
		for _, r := range roles {
			if set[r] {
				return c.Next()
			}
		}
		return forbiddenResponse(c, "role")
	}
}

// ─── Gate-based middleware (resource authorization) ─────────────────────────

// GateAllows returns middleware that checks a gate ability without a resource.
// Suitable for collection-level checks (e.g. "viewAny").
//
// Usage:
//
//	app.Get("/posts", middleware.GateAllows(gate, "post.viewAny"), handler)
func GateAllows(gate *auth.Gate, ability string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if gate.Denies(c, ability, nil) {
			return forbiddenResponse(c, ability)
		}
		return c.Next()
	}
}

// ─── Inject user roles middleware ──────────────────────────────────────────

// InjectRoles loads user roles from RBAC into Locals so that Can/HasRole middleware
// can read them without a DB call per middleware. Place this after authentication
// middleware that sets LocalsUserID.
//
// Usage:
//
//	app.Use(security.JWTMiddleware(cfg))
//	app.Use(middleware.InjectRoles(rbac))
//	app.Get("/admin", middleware.Can(rbac, "admin.access"), handler)
func InjectRoles(rbac *auth.RBACManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, ok := c.Locals(LocalsUserID).(uint)
		if !ok {
			// Try float64 (common when coming from JWT MapClaims).
			if fid, fok := c.Locals(LocalsUserID).(float64); fok {
				userID = uint(fid)
				ok = true
			}
		}
		if !ok || userID == 0 {
			return c.Next()
		}
		roles, err := rbac.UserRoles(c.Context(), userID)
		if err != nil {
			return c.Next() // Graceful: proceed without roles.
		}
		c.Locals(LocalsUserRoles, roles)
		return c.Next()
	}
}

// ─── helpers ───────────────────────────────────────────────────────────────

func forbiddenResponse(c fiber.Ctx, permission string) error {
	msg := i18n.T(c, "auth.forbidden")
	if msg == "auth.forbidden" {
		msg = "You do not have permission to perform this action."
	}
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"error": fiber.Map{
			"code":       fiber.StatusForbidden,
			"message":    msg,
			"permission": permission,
		},
	})
}
