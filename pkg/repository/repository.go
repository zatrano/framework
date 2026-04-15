package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Repository is a generic CRUD repository interface for any GORM model.
// T must be a struct type (e.g. User, Product).
//
//	type UserRepository interface {
//	    repository.Repository[User]
//	    FindByEmail(ctx context.Context, email string) (*User, error)
//	}
type Repository[T any] interface {
	// FindByID returns a single record by primary key.
	FindByID(ctx context.Context, id uint, scopes ...Scope) (*T, error)

	// FindAll returns all records matching the given scopes.
	FindAll(ctx context.Context, scopes ...Scope) ([]T, error)

	// Paginate returns a page of records.
	Paginate(ctx context.Context, opts PaginateOpts, scopes ...Scope) (Page[T], error)

	// First returns the first record matching the scopes.
	First(ctx context.Context, scopes ...Scope) (*T, error)

	// Create inserts a new record.
	Create(ctx context.Context, model *T) error

	// Update saves all changed fields of the given record.
	Update(ctx context.Context, model *T) error

	// Save upserts the record (insert or update based on primary key).
	Save(ctx context.Context, model *T) error

	// Delete soft-deletes the record.
	Delete(ctx context.Context, model *T) error

	// DeleteByID soft-deletes a record by primary key.
	DeleteByID(ctx context.Context, id uint) error

	// HardDelete permanently removes the record from the database.
	HardDelete(ctx context.Context, model *T) error

	// Restore un-deletes a soft-deleted record.
	Restore(ctx context.Context, id uint) error

	// Count returns the number of records matching the scopes.
	Count(ctx context.Context, scopes ...Scope) (int64, error)

	// Exists reports whether at least one record matches the scopes.
	Exists(ctx context.Context, scopes ...Scope) (bool, error)

	// DB returns the underlying *gorm.DB for custom queries.
	DB() *gorm.DB
}

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("record not found")

// ─── GORM Implementation ───────────────────────────────────────────────────

// GormRepository is the default GORM-backed implementation of Repository[T].
type GormRepository[T any] struct {
	db *gorm.DB
}

// New creates a new GormRepository for the given model type.
//
//	userRepo := repository.New[User](app.DB)
func New[T any](db *gorm.DB) *GormRepository[T] {
	return &GormRepository[T]{db: db}
}

// Ensure interface compliance at compile time.
var _ Repository[struct{}] = (*GormRepository[struct{}])(nil)

func (r *GormRepository[T]) DB() *gorm.DB { return r.db }

func (r *GormRepository[T]) base(scopes []Scope) *gorm.DB {
	return applyScopes(r.db, scopes)
}

// FindByID returns a single record by primary key.
func (r *GormRepository[T]) FindByID(ctx context.Context, id uint, scopes ...Scope) (*T, error) {
	var model T
	result := r.base(scopes).WithContext(ctx).First(&model, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("FindByID(%d): %w", id, ErrNotFound)
	}
	return &model, result.Error
}

// FindAll returns all records matching the given scopes.
func (r *GormRepository[T]) FindAll(ctx context.Context, scopes ...Scope) ([]T, error) {
	var models []T
	result := r.base(scopes).WithContext(ctx).Find(&models)
	return models, result.Error
}

// Paginate returns a page of records with count metadata.
func (r *GormRepository[T]) Paginate(ctx context.Context, opts PaginateOpts, scopes ...Scope) (Page[T], error) {
	opts.Normalize()

	var total int64
	var model T

	base := r.base(scopes).WithContext(ctx)
	if err := base.Model(&model).Count(&total).Error; err != nil {
		return Page[T]{}, err
	}

	var items []T
	if err := base.Offset(opts.Offset()).Limit(opts.PerPage).Find(&items).Error; err != nil {
		return Page[T]{}, err
	}

	return Page[T]{
		Items:      items,
		Pagination: NewPaginationMeta(opts.Page, opts.PerPage, total),
	}, nil
}

// First returns the first matching record.
func (r *GormRepository[T]) First(ctx context.Context, scopes ...Scope) (*T, error) {
	var model T
	result := r.base(scopes).WithContext(ctx).First(&model)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &model, result.Error
}

// Create inserts a new record.
func (r *GormRepository[T]) Create(ctx context.Context, model *T) error {
	return r.db.WithContext(ctx).Create(model).Error
}

// Update saves all changed fields of the given record.
func (r *GormRepository[T]) Update(ctx context.Context, model *T) error {
	return r.db.WithContext(ctx).Save(model).Error
}

// Save upserts the record.
func (r *GormRepository[T]) Save(ctx context.Context, model *T) error {
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete soft-deletes the record.
func (r *GormRepository[T]) Delete(ctx context.Context, model *T) error {
	return r.db.WithContext(ctx).Delete(model).Error
}

// DeleteByID soft-deletes a record by primary key.
func (r *GormRepository[T]) DeleteByID(ctx context.Context, id uint) error {
	var model T
	return r.db.WithContext(ctx).Delete(&model, id).Error
}

// HardDelete permanently removes the record.
func (r *GormRepository[T]) HardDelete(ctx context.Context, model *T) error {
	return r.db.WithContext(ctx).Unscoped().Delete(model).Error
}

// Restore un-deletes a soft-deleted record.
func (r *GormRepository[T]) Restore(ctx context.Context, id uint) error {
	var model T
	return r.db.WithContext(ctx).Unscoped().Model(&model).Where("id = ?", id).
		Update("deleted_at", nil).Error
}

// Count returns the number of records matching the scopes.
func (r *GormRepository[T]) Count(ctx context.Context, scopes ...Scope) (int64, error) {
	var model T
	var count int64
	err := r.base(scopes).WithContext(ctx).Model(&model).Count(&count).Error
	return count, err
}

// Exists reports whether at least one record matches.
func (r *GormRepository[T]) Exists(ctx context.Context, scopes ...Scope) (bool, error) {
	count, err := r.Count(ctx, scopes...)
	return count > 0, err
}

// ─── Transaction helper ───────────────────────────────────────────────────

// Tx creates a new GormRepository scoped to a transaction.
//
//	err := app.DB.Transaction(func(tx *gorm.DB) error {
//	    userRepo := repository.New[User](tx)
//	    return userRepo.Create(ctx, &user)
//	})
func (r *GormRepository[T]) Tx(tx *gorm.DB) *GormRepository[T] {
	return &GormRepository[T]{db: tx}
}
