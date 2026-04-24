// Package repositories içindeki BaseRepository güvenlik güncellemesi.
//
// Bu dosya mevcut repositories/base_repository.go'nun GÜVENLİ versiyonudur.
// Değişiklikler:
//  1. SetAllowedSortColumns zorunlu hale getirildi
//  2. Sıralama kolonları allowlist'ten geçiriliyor
//  3. Preload desteği eklendi (N+1 önleme)
//  4. Sayfalama limiti eklendi (DoS önleme)
//  5. SearchValue LIKE escape eklendi
//
// KULLANIM — mevcut base_repository.go'yu bu içerikle GÜNCELLEYİN.

package repositories

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/zatrano/framework/packages/eagerload"
	"github.com/zatrano/framework/packages/queryparams"
	"github.com/zatrano/framework/packages/safequery"
)

const (
	// DefaultPageSize varsayılan sayfa boyutu.
	DefaultPageSize = 20
	// MaxPageSize maksimum sayfa boyutu (DoS önleme).
	MaxPageSize = 100
	// DefaultSortColumn varsayılan sıralama kolonu.
	DefaultSortColumn = "id"
	// DefaultSortDirection varsayılan sıralama yönü.
	DefaultSortDirection = "DESC"
)

// IBaseRepository tüm repository'lerin implement etmesi gereken temel arayüz.
type IBaseRepository[T any] interface {
	GetAll(ctx context.Context, params queryparams.ListParams, preloads ...eagerload.PreloadOption) ([]T, int64, error)
	GetByID(ctx context.Context, id uint, preloads ...eagerload.PreloadOption) (*T, error)
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error
	Delete(ctx context.Context, id uint) error
	Exists(ctx context.Context, id uint) (bool, error)
	GetCount(ctx context.Context) (int64, error)
}

// BaseRepository jenerik temel repository implementasyonu.
type BaseRepository[T any] struct {
	db                  *gorm.DB
	allowedSortColumns  []string
}

// NewBaseRepository yeni bir BaseRepository oluşturur.
func NewBaseRepository[T any](db *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{
		db:                 db,
		allowedSortColumns: []string{DefaultSortColumn, "created_at", "updated_at"},
	}
}

// DB veritabanı bağlantısını döner (alt sınıflarda özel sorgular için).
func (r *BaseRepository[T]) DB() *gorm.DB {
	return r.db
}

// SetAllowedSortColumns sıralamaya izin verilen kolon adlarını belirler.
// Repository init'inde MUTLAKA çağrılmalıdır.
// Bu liste dışındaki kolonlarla gelen sıralama istekleri varsayılana döner.
//
// Örnek:
//
//	repo.SetAllowedSortColumns([]string{"id", "name", "price", "created_at"})
func (r *BaseRepository[T]) SetAllowedSortColumns(cols []string) {
	r.allowedSortColumns = cols
}

// GetAll güvenli sayfalama, sıralama ve preload ile kayıtları listeler.
//
// GÜVENLİK:
//   - Sıralama kolonu allowlist'ten doğrulanır (SQL Injection önleme)
//   - Sıralama yönü yalnızca ASC/DESC kabul edilir
//   - Sayfa boyutu MaxPageSize ile sınırlandırılır (DoS önleme)
//   - Preload'lar N+1 sorununu önler
//
// Örnek:
//
//	products, total, err := repo.GetAll(ctx, params, eagerload.Opt("Category"), eagerload.Opt("Images"))
func (r *BaseRepository[T]) GetAll(
	ctx context.Context,
	params queryparams.ListParams,
	preloads ...eagerload.PreloadOption,
) ([]T, int64, error) {
	var entities []T
	var total int64

	// Güvenli sayfalama (DoS önleme)
	page, limit := safequery.Paginate(params.Page, params.PerPage, DefaultPageSize, MaxPageSize)
	offset := (page - 1) * limit

	// Güvenli sıralama — kolon allowlist'ten doğrulanır
	sortCol, err := safequery.AllowedColumns(params.SortBy, r.allowedSortColumns)
	if err != nil {
		sortCol = DefaultSortColumn // geçersizse varsayılana dön
	}
	sortDir, err := safequery.ValidateDirection(params.OrderBy)
	if err != nil {
		sortDir = DefaultSortDirection
	}
	orderClause := sortCol + " " + sortDir

	// Temel sorgu
	db := r.db.WithContext(ctx).Model(new(T))

	// Arama filtresi (LIKE — güvenli escape)
	if params.Search != "" {
		safeSearch := safequery.SearchValue(params.Search)
		db = db.Where("name ILIKE ?", "%"+safeSearch+"%") // ILIKE = PostgreSQL case-insensitive
	}

	// Toplam kayıt sayısı (preload olmadan — performans)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Preload uygula (N+1 önleme)
	db = eagerload.Apply(db, preloads...)

	// Sıralama ve sayfalama
	if err := db.Order(orderClause).Limit(limit).Offset(offset).Find(&entities).Error; err != nil {
		return nil, 0, err
	}

	return entities, total, nil
}

// GetByID ID ile tek kayıt çeker, preload destekler.
//
// GÜVENLİK: ID uint tipinde alınır (negatif değer ve string injection imkansız).
//
// Örnek:
//
//	product, err := repo.GetByID(ctx, id, eagerload.Opt("Category"))
func (r *BaseRepository[T]) GetByID(
	ctx context.Context,
	id uint,
	preloads ...eagerload.PreloadOption,
) (*T, error) {
	var entity T
	db := r.db.WithContext(ctx)
	db = eagerload.Apply(db, preloads...)
	if err := db.First(&entity, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &entity, nil
}

// Create yeni kayıt oluşturur.
// Veriler handler/service katmanında sanitize edilmiş olmalıdır.
func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// Update mevcut kaydı günceller.
// Veriler handler/service katmanında sanitize edilmiş olmalıdır.
func (r *BaseRepository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

// UpdateFields — yalnızca verilen alanları (map) günceller; kısmi CRUD için.
func (r *BaseRepository[T]) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(new(T)).Where("id = ?", id).Updates(fields).Error
}

// GetCount — tablodaki toplam kayıt sayısını döner.
func (r *BaseRepository[T]) GetCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(new(T)).Count(&count).Error
	return count, err
}

// Delete soft delete yapar (BaseModel.DeletedAt doldurulur, kayıt silinmez).
// Hard delete için: r.db.Unscoped().Delete(&entity, id)
func (r *BaseRepository[T]) Delete(ctx context.Context, id uint) error {
	var entity T
	return r.db.WithContext(ctx).Delete(&entity, id).Error
}

// Exists ID'li kaydın var olup olmadığını kontrol eder.
// COUNT(*) kullanır — veri çekmez, performanslıdır.
func (r *BaseRepository[T]) Exists(ctx context.Context, id uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(new(T)).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

// WithTx transaction context'iyle yeni bir repository döner.
// txmanager.WithTransaction içinde kullanılır.
//
// Örnek:
//
//	txmanager.WithTransaction(ctx, func(txCtx context.Context) error {
//	    txRepo := repo.WithTx(txCtx)
//	    return txRepo.Create(txCtx, &entity)
//	})
func (r *BaseRepository[T]) WithTx(db *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{
		db:                 db,
		allowedSortColumns: r.allowedSortColumns,
	}
}

// SearchByField belirli bir kolonda parametreli arama yapar.
//
// GÜVENLİK: fieldName allowlist'ten doğrulanır.
//
// Örnek:
//
//	users, total, err := repo.SearchByField(ctx, "email", "test@", params)
func (r *BaseRepository[T]) SearchByField(
	ctx context.Context,
	fieldName string,
	value string,
	params queryparams.ListParams,
) ([]T, int64, error) {
	// Kolon adını güvenlik için doğrula
	safeField, err := safequery.AllowedColumns(fieldName, r.allowedSortColumns)
	if err != nil {
		return nil, 0, errors.New("geçersiz alan adı: " + fieldName)
	}

	var entities []T
	var total int64

	safeVal := safequery.SearchValue(value)
	// Parametreli sorgu — SQL injection imkansız
	condition := strings.ToLower(safeField) + " ILIKE ?"
	searchVal := "%" + safeVal + "%"

	db := r.db.WithContext(ctx).Model(new(T)).Where(condition, searchVal)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, limit := safequery.Paginate(params.Page, params.PerPage, DefaultPageSize, MaxPageSize)
	offset := (page - 1) * limit

	sortCol, _ := safequery.AllowedColumns(params.SortBy, r.allowedSortColumns)
	if sortCol == "" {
		sortCol = DefaultSortColumn
	}
	sortDir, _ := safequery.ValidateDirection(params.OrderBy)
	if sortDir == "" {
		sortDir = DefaultSortDirection
	}

	if err := db.Order(sortCol+" "+sortDir).Limit(limit).Offset(offset).Find(&entities).Error; err != nil {
		return nil, 0, err
	}

	return entities, total, nil
}

// ErrNotFound kayıt bulunamadı hatası.
var ErrNotFound = errors.New("kayıt bulunamadı")
