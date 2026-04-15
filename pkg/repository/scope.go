package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Scope is a function that modifies a gorm.DB query.
// Scopes can be composed to build complex, reusable query logic.
//
//	repo.FindAll(ctx, scope.Active(), scope.OrderBy("created_at DESC"))
type Scope func(*gorm.DB) *gorm.DB

// Scopes is a helper to apply multiple Scope functions at once.
func Scopes(scopes ...Scope) []Scope { return scopes }

// ─── Built-in Scopes ────────────────────────────────────────────────────────

// Active filters records where the `active` column is true.
func Active() Scope { return func(db *gorm.DB) *gorm.DB { return db.Where("active = ?", true) } }

// Inactive filters records where the `active` column is false.
func Inactive() Scope { return func(db *gorm.DB) *gorm.DB { return db.Where("active = ?", false) } }

// OrderBy applies an ORDER BY clause.
//
//	scope.OrderBy("created_at DESC")
func OrderBy(column string) Scope {
	return func(db *gorm.DB) *gorm.DB { return db.Order(column) }
}

// Limit restricts the number of returned records.
func Limit(n int) Scope {
	return func(db *gorm.DB) *gorm.DB { return db.Limit(n) }
}

// Where adds a WHERE condition.
//
//	scope.Where("email = ?", "alice@example.com")
func Where(query any, args ...any) Scope {
	return func(db *gorm.DB) *gorm.DB { return db.Where(query, args...) }
}

// WithTrashed includes soft-deleted records in results.
func WithTrashed() Scope {
	return func(db *gorm.DB) *gorm.DB { return db.Unscoped() }
}

// OnlyTrashed returns only soft-deleted records.
func OnlyTrashed() Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Unscoped().Where("deleted_at IS NOT NULL")
	}
}

// Preload eagerly loads a named association.
//
//	scope.Preload("Profile")
//	scope.Preload("Orders", "status = ?", "active")
func Preload(association string, args ...any) Scope {
	return func(db *gorm.DB) *gorm.DB { return db.Preload(association, args...) }
}

// PreloadAll eagerly loads all first-level associations.
func PreloadAll() Scope {
	return func(db *gorm.DB) *gorm.DB { return db.Preload(clause.Associations) }
}

// Select limits the columns returned.
//
//	scope.Select("id", "name", "email")
func Select(columns ...string) Scope {
	return func(db *gorm.DB) *gorm.DB { return db.Select(columns) }
}

// ─── Scope Builder ─────────────────────────────────────────────────────────

// Builder chains scopes into a single gorm.DB.
func applyScopes(db *gorm.DB, scopes []Scope) *gorm.DB {
	for _, s := range scopes {
		if s != nil {
			db = s(db)
		}
	}
	return db
}

// ─── Scope from context ─────────────────────────────────────────────────────

type contextKey string

const scopeKey contextKey = "repository_scopes"

// WithScopes stores scopes in the context for use by middleware or services.
func WithScopes(ctx context.Context, scopes ...Scope) context.Context {
	existing, _ := ctx.Value(scopeKey).([]Scope)
	return context.WithValue(ctx, scopeKey, append(existing, scopes...))
}

// ScopesFromContext retrieves scopes stored in the context.
func ScopesFromContext(ctx context.Context) []Scope {
	scopes, _ := ctx.Value(scopeKey).([]Scope)
	return scopes
}
