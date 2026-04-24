package repositories

import (
	"context"
	"strings"

	"github.com/zatrano/framework/configs/databaseconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/requests"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IDefinitionRepository interface {
	ListAll(ctx context.Context) ([]models.Definition, error)
	ListPaged(ctx context.Context, params requests.DefinitionListParams) ([]models.Definition, int64, error)
	GetByID(ctx context.Context, id uint) (*models.Definition, error)
	GetByKey(ctx context.Context, key string) (*models.Definition, error)
	GetByKeyAny(ctx context.Context, key string) (*models.Definition, error)
	GetByKeySlugs(ctx context.Context, keys []string) ([]models.Definition, error)
	Upsert(ctx context.Context, d *models.Definition) error
	Create(ctx context.Context, d *models.Definition) error
	UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error
	Delete(ctx context.Context, id uint) error
}

type DefinitionRepository struct {
	db *gorm.DB
}

func NewDefinitionRepository() IDefinitionRepository {
	return &DefinitionRepository{db: databaseconfig.GetDB()}
}

func (r *DefinitionRepository) ListAll(ctx context.Context) ([]models.Definition, error) {
	var defs []models.Definition
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("key ASC").Find(&defs).Error; err != nil {
		return nil, err
	}
	return defs, nil
}

func (r *DefinitionRepository) ListPaged(ctx context.Context, params requests.DefinitionListParams) ([]models.Definition, int64, error) {
	var defs []models.Definition
	var total int64
	q := r.db.WithContext(ctx).Model(&models.Definition{})
	if params.Key != "" {
		pat := "%" + params.Key + "%"
		q = q.Where("key ILIKE ? OR description ILIKE ?", pat, pat)
	}
	if params.IsActive != "" {
		switch params.IsActive {
		case "true":
			q = q.Where("is_active = ?", true)
		case "false":
			q = q.Where("is_active = ?", false)
		}
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []models.Definition{}, 0, nil
	}
	orderCol := params.SortBy
	switch orderCol {
	case "id", "key", "created_at":
	default:
		orderCol = "key"
	}
	dir := strings.ToUpper(params.OrderBy)
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	if err := q.Order(orderCol + " " + dir).Limit(params.PerPage).Offset(params.CalculateOffset()).Find(&defs).Error; err != nil {
		return nil, 0, err
	}
	return defs, total, nil
}

func (r *DefinitionRepository) GetByID(ctx context.Context, id uint) (*models.Definition, error) {
	var d models.Definition
	if err := r.db.WithContext(ctx).First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DefinitionRepository) GetByKeyAny(ctx context.Context, key string) (*models.Definition, error) {
	var d models.Definition
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DefinitionRepository) GetByKey(ctx context.Context, key string) (*models.Definition, error) {
	var d models.Definition
	err := r.db.WithContext(ctx).
		Where("key = ? AND is_active = ?", key, true).
		First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DefinitionRepository) GetByKeySlugs(ctx context.Context, keys []string) ([]models.Definition, error) {
	if len(keys) == 0 {
		return []models.Definition{}, nil
	}
	var defs []models.Definition
	err := r.db.WithContext(ctx).
		Where("key IN ? AND is_active = ?", keys, true).
		Find(&defs).Error
	if err != nil {
		return nil, err
	}
	return defs, nil
}

func (r *DefinitionRepository) Upsert(ctx context.Context, d *models.Definition) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "description", "updated_at", "updated_by", "is_active"}),
	}).Create(d).Error
}

func (r *DefinitionRepository) Create(ctx context.Context, d *models.Definition) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *DefinitionRepository) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Definition{}).Where("id = ?", id).Updates(fields).Error
}

func (r *DefinitionRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Definition{}, id).Error
}
