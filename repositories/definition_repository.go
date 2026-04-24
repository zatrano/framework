package repositories

import (
	"context"

	"github.com/zatrano/framework/configs/databaseconfig"
	"github.com/zatrano/framework/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IDefinitionRepository interface {
	ListAll(ctx context.Context) ([]models.Definition, error)
	GetByKey(ctx context.Context, key string) (*models.Definition, error)
	GetByKeySlugs(ctx context.Context, keys []string) ([]models.Definition, error)
	Upsert(ctx context.Context, d *models.Definition) error
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
