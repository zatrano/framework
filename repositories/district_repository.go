package repositories

import (
	"context"

	"github.com/zatrano/framework/configs/databaseconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/requests"

	"gorm.io/gorm"
)

type IDistrictRepository interface {
	GetAllDistricts(ctx context.Context, params requests.DistrictListParams) ([]models.District, int64, error)
	GetDistrictByID(ctx context.Context, id uint) (*models.District, error)
	CreateDistrict(ctx context.Context, district *models.District) error
	UpdateDistrict(ctx context.Context, id uint, data map[string]interface{}) error
	DeleteDistrict(ctx context.Context, id uint) error
	GetDistrictCount(ctx context.Context) (int64, error)
}

type DistrictRepository struct {
	base IBaseRepository[models.District]
	db   *gorm.DB
}

func NewDistrictRepository() IDistrictRepository {
	base := NewBaseRepository[models.District](databaseconfig.GetDB())
	base.SetAllowedSortColumns([]string{"id", "name", "created_at"})
	return &DistrictRepository{base: base, db: databaseconfig.GetDB()}
}

func (r *DistrictRepository) GetAllDistricts(ctx context.Context, params requests.DistrictListParams) ([]models.District, int64, error) {
	var districts []models.District
	var totalCount int64
	query := r.db.WithContext(ctx).Model(&models.District{}).Preload("Country").Preload("City")
	if params.CountryID != nil {
		query = query.Where("country_id = ?", *params.CountryID)
	}
	if params.CityID != nil {
		query = query.Where("city_id = ?", *params.CityID)
	}
	if params.Name != "" {
		query = query.Where("name ILIKE ?", "%"+params.Name+"%")
	}
	if params.IsActive != "" {
		switch params.IsActive {
		case "true":
			query = query.Where("is_active = ?", true)
		case "false":
			query = query.Where("is_active = ?", false)
		}
	}
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}
	if totalCount == 0 {
		return []models.District{}, 0, nil
	}
	query = query.Order(params.SortBy + " " + params.OrderBy).Limit(params.PerPage).Offset(params.CalculateOffset())
	if err := query.Find(&districts).Error; err != nil {
		return nil, 0, err
	}
	return districts, totalCount, nil
}

func (r *DistrictRepository) GetDistrictByID(ctx context.Context, id uint) (*models.District, error) {
	var district models.District
	err := r.db.WithContext(ctx).Preload("Country").Preload("City").First(&district, id).Error
	if err != nil {
		return nil, err
	}
	return &district, nil
}

func (r *DistrictRepository) CreateDistrict(ctx context.Context, district *models.District) error {
	return r.base.Create(ctx, district)
}

func (r *DistrictRepository) UpdateDistrict(ctx context.Context, id uint, data map[string]interface{}) error {
	return r.base.UpdateFields(ctx, id, data)
}

func (r *DistrictRepository) DeleteDistrict(ctx context.Context, id uint) error {
	return r.base.Delete(ctx, id)
}

func (r *DistrictRepository) GetDistrictCount(ctx context.Context) (int64, error) {
	return r.base.GetCount(ctx)
}
