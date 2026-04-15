package auth

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// ─── GORM Models ────────────────────────────────────────────────────────────

// Role represents an RBAC role stored in the database.
type Role struct {
	ID          uint         `gorm:"primaryKey" json:"id"`
	Name        string       `gorm:"uniqueIndex;size:100;not null" json:"name"`
	Description string       `gorm:"size:255" json:"description,omitempty"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Permission represents a single permission (e.g. "posts.create", "users.delete").
type Permission struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"uniqueIndex;size:100;not null" json:"name"`
	Description string `gorm:"size:255" json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserRole is the join table between users and roles.
// The "user_id" column type should match your users table PK; uint covers most cases.
type UserRole struct {
	UserID uint `gorm:"primaryKey" json:"user_id"`
	RoleID uint `gorm:"primaryKey;index" json:"role_id"`
}

// TableName overrides the default "user_roles" to "zatrano_user_roles".
func (UserRole) TableName() string { return "zatrano_user_roles" }

// ─── RBAC Manager ───────────────────────────────────────────────────────────

// RBACManager provides role and permission operations backed by GORM.
// It keeps an in-memory cache (rolePerms) so hot-path checks avoid DB hits.
type RBACManager struct {
	db *gorm.DB

	mu        sync.RWMutex
	rolePerms map[string]map[string]bool // role name → set of permission names
}

// NewRBACManager creates a manager and warms the cache.
func NewRBACManager(db *gorm.DB) (*RBACManager, error) {
	m := &RBACManager{
		db:        db,
		rolePerms: make(map[string]map[string]bool),
	}
	if err := m.Refresh(context.Background()); err != nil {
		return nil, fmt.Errorf("rbac: initial cache refresh: %w", err)
	}
	return m, nil
}

// ─── Cache ──────────────────────────────────────────────────────────────────

// Refresh reloads the full role→permission map from the database.
func (m *RBACManager) Refresh(ctx context.Context) error {
	var roles []Role
	if err := m.db.WithContext(ctx).Preload("Permissions").Find(&roles).Error; err != nil {
		return err
	}
	built := make(map[string]map[string]bool, len(roles))
	for _, r := range roles {
		perms := make(map[string]bool, len(r.Permissions))
		for _, p := range r.Permissions {
			perms[p.Name] = true
		}
		built[r.Name] = perms
	}
	m.mu.Lock()
	m.rolePerms = built
	m.mu.Unlock()
	return nil
}

// ─── Permission checks (cache-based, no DB hit) ────────────────────────────

// RoleHasPermission checks whether a role has a specific permission (cached).
func (m *RBACManager) RoleHasPermission(role, permission string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	perms, ok := m.rolePerms[role]
	if !ok {
		return false
	}
	return perms[permission]
}

// RolesHavePermission returns true if ANY of the given roles has the permission.
func (m *RBACManager) RolesHavePermission(roles []string, permission string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range roles {
		if m.rolePerms[r][permission] {
			return true
		}
	}
	return false
}

// RolesHaveAnyPermission returns true if any role owns any of the supplied permissions.
func (m *RBACManager) RolesHaveAnyPermission(roles []string, permissions ...string) bool {
	for _, p := range permissions {
		if m.RolesHavePermission(roles, p) {
			return true
		}
	}
	return false
}

// RolesHaveAllPermissions returns true only when the union of all roles covers every permission.
func (m *RBACManager) RolesHaveAllPermissions(roles []string, permissions ...string) bool {
	for _, p := range permissions {
		if !m.RolesHavePermission(roles, p) {
			return false
		}
	}
	return true
}

// ─── Role CRUD ──────────────────────────────────────────────────────────────

// CreateRole persists a new role and refreshes the cache.
func (m *RBACManager) CreateRole(ctx context.Context, name, description string) (*Role, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("rbac: role name is required")
	}
	r := &Role{Name: name, Description: description}
	if err := m.db.WithContext(ctx).Create(r).Error; err != nil {
		return nil, fmt.Errorf("rbac: create role: %w", err)
	}
	_ = m.Refresh(ctx)
	return r, nil
}

// DeleteRole removes a role and its associations, then refreshes the cache.
func (m *RBACManager) DeleteRole(ctx context.Context, name string) error {
	var role Role
	if err := m.db.WithContext(ctx).Where("name = ?", name).First(&role).Error; err != nil {
		return fmt.Errorf("rbac: role not found: %w", err)
	}
	if err := m.db.WithContext(ctx).Model(&role).Association("Permissions").Clear(); err != nil {
		return fmt.Errorf("rbac: clear permissions: %w", err)
	}
	if err := m.db.WithContext(ctx).Delete(&role).Error; err != nil {
		return fmt.Errorf("rbac: delete role: %w", err)
	}
	_ = m.Refresh(ctx)
	return nil
}

// ─── Permission CRUD ────────────────────────────────────────────────────────

// CreatePermission persists a new permission and refreshes the cache.
func (m *RBACManager) CreatePermission(ctx context.Context, name, description string) (*Permission, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("rbac: permission name is required")
	}
	p := &Permission{Name: name, Description: description}
	if err := m.db.WithContext(ctx).Create(p).Error; err != nil {
		return nil, fmt.Errorf("rbac: create permission: %w", err)
	}
	return p, nil
}

// ─── Role ↔ Permission assignment ──────────────────────────────────────────

// AssignPermissions attaches permissions to a role (by names).
func (m *RBACManager) AssignPermissions(ctx context.Context, roleName string, permissionNames ...string) error {
	var role Role
	if err := m.db.WithContext(ctx).Where("name = ?", roleName).First(&role).Error; err != nil {
		return fmt.Errorf("rbac: role %q not found: %w", roleName, err)
	}
	var perms []Permission
	if err := m.db.WithContext(ctx).Where("name IN ?", permissionNames).Find(&perms).Error; err != nil {
		return fmt.Errorf("rbac: find permissions: %w", err)
	}
	if len(perms) == 0 {
		return fmt.Errorf("rbac: no matching permissions found")
	}
	if err := m.db.WithContext(ctx).Model(&role).Association("Permissions").Append(&perms); err != nil {
		return fmt.Errorf("rbac: assign permissions: %w", err)
	}
	_ = m.Refresh(ctx)
	return nil
}

// RevokePermissions detaches permissions from a role.
func (m *RBACManager) RevokePermissions(ctx context.Context, roleName string, permissionNames ...string) error {
	var role Role
	if err := m.db.WithContext(ctx).Where("name = ?", roleName).First(&role).Error; err != nil {
		return fmt.Errorf("rbac: role %q not found: %w", roleName, err)
	}
	var perms []Permission
	if err := m.db.WithContext(ctx).Where("name IN ?", permissionNames).Find(&perms).Error; err != nil {
		return fmt.Errorf("rbac: find permissions: %w", err)
	}
	if err := m.db.WithContext(ctx).Model(&role).Association("Permissions").Delete(&perms); err != nil {
		return fmt.Errorf("rbac: revoke permissions: %w", err)
	}
	_ = m.Refresh(ctx)
	return nil
}

// ─── User ↔ Role assignment ────────────────────────────────────────────────

// AssignRoleToUser links a user to a role.
func (m *RBACManager) AssignRoleToUser(ctx context.Context, userID uint, roleName string) error {
	var role Role
	if err := m.db.WithContext(ctx).Where("name = ?", roleName).First(&role).Error; err != nil {
		return fmt.Errorf("rbac: role %q not found: %w", roleName, err)
	}
	ur := UserRole{UserID: userID, RoleID: role.ID}
	if err := m.db.WithContext(ctx).Where(ur).FirstOrCreate(&ur).Error; err != nil {
		return fmt.Errorf("rbac: assign role: %w", err)
	}
	return nil
}

// RemoveRoleFromUser removes a user→role association.
func (m *RBACManager) RemoveRoleFromUser(ctx context.Context, userID uint, roleName string) error {
	var role Role
	if err := m.db.WithContext(ctx).Where("name = ?", roleName).First(&role).Error; err != nil {
		return fmt.Errorf("rbac: role %q not found: %w", roleName, err)
	}
	if err := m.db.WithContext(ctx).Where("user_id = ? AND role_id = ?", userID, role.ID).Delete(&UserRole{}).Error; err != nil {
		return fmt.Errorf("rbac: remove role: %w", err)
	}
	return nil
}

// UserRoles returns the role names assigned to a user.
func (m *RBACManager) UserRoles(ctx context.Context, userID uint) ([]string, error) {
	var roles []Role
	err := m.db.WithContext(ctx).
		Joins("JOIN zatrano_user_roles ur ON ur.role_id = roles.id").
		Where("ur.user_id = ?", userID).
		Find(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("rbac: user roles: %w", err)
	}
	names := make([]string, len(roles))
	for i, r := range roles {
		names[i] = r.Name
	}
	return names, nil
}

// UserHasPermission loads user roles from DB and checks the cache in one call.
func (m *RBACManager) UserHasPermission(ctx context.Context, userID uint, permission string) (bool, error) {
	roles, err := m.UserRoles(ctx, userID)
	if err != nil {
		return false, err
	}
	return m.RolesHavePermission(roles, permission), nil
}

// UserHasRole checks whether the user is assigned to the given role.
func (m *RBACManager) UserHasRole(ctx context.Context, userID uint, roleName string) (bool, error) {
	roles, err := m.UserRoles(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r == roleName {
			return true, nil
		}
	}
	return false, nil
}

// ─── Listing helpers ────────────────────────────────────────────────────────

// AllRoles returns all roles with their permissions.
func (m *RBACManager) AllRoles(ctx context.Context) ([]Role, error) {
	var roles []Role
	if err := m.db.WithContext(ctx).Preload("Permissions").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// AllPermissions returns every permission.
func (m *RBACManager) AllPermissions(ctx context.Context) ([]Permission, error) {
	var perms []Permission
	if err := m.db.WithContext(ctx).Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

// RolePermissions returns permission names for a role (from cache).
func (m *RBACManager) RolePermissions(roleName string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	perms := m.rolePerms[roleName]
	out := make([]string, 0, len(perms))
	for p := range perms {
		out = append(out, p)
	}
	return out
}

// ─── Auto-migrate helper ───────────────────────────────────────────────────

// AutoMigrate creates or updates RBAC tables (roles, permissions, role_permissions, zatrano_user_roles).
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Role{}, &Permission{}, &UserRole{})
}
