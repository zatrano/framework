package repositories

import (
	"context"

	"github.com/zatrano/framework/configs/databaseconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/requests"

	"gorm.io/gorm"
)

type ICityRepository interface {
	GetAllCities(ctx context.Context, params requests.CityListParams) ([]models.City, int64, error)
	GetCityByID(ctx context.Context, id uint) (*models.City, error)
	CreateCity(ctx context.Context, city *models.City) error
	UpdateCity(ctx context.Context, id uint, data map[string]interface{}) error
	DeleteCity(ctx context.Context, id uint) error
	GetCityCount(ctx context.Context) (int64, error)
}

type CityRepository struct {
	base IBaseRepository[models.City]
	db   *gorm.DB
}

func NewCityRepository() ICityRepository {
	base := NewBaseRepository[models.City](databaseconfig.GetDB())
	base.SetAllowedSortColumns([]string{"id", "name", "created_at"})
	return &CityRepository{base: base, db: databaseconfig.GetDB()}
}

func (r *CityRepository) GetAllCities(ctx context.Context, params requests.CityListParams) ([]models.City, int64, error) {
	var cities []models.City
	var totalCount int64
	query := r.db.WithContext(ctx).Model(&models.City{}).Preload("Country")
	if params.CountryID != nil {
		query = query.Where("country_id = ?", *params.CountryID)
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
		return []models.City{}, 0, nil
	}
	query = query.Order(params.SortBy + " " + params.OrderBy).Limit(params.PerPage).Offset(params.CalculateOffset())
	if err := query.Find(&cities).Error; err != nil {
		return nil, 0, err
	}
	return cities, totalCount, nil
}

func (r *CityRepository) GetCityByID(ctx context.Context, id uint) (*models.City, error) {
	var city models.City
	err := r.db.WithContext(ctx).Preload("Country").First(&city, id).Error
	if err != nil {
		return nil, err
	}
	return &city, nil
}

func (r *CityRepository) CreateCity(ctx context.Context, city *models.City) error {
	return r.base.Create(ctx, city)
}

func (r *CityRepository) UpdateCity(ctx context.Context, id uint, data map[string]interface{}) error {
	return r.base.UpdateFields(ctx, id, data)
}

func (r *CityRepository) DeleteCity(ctx context.Context, id uint) error {
	return r.base.Delete(ctx, id)
}

func (r *CityRepository) GetCityCount(ctx context.Context) (int64, error) {
	return r.base.GetCount(ctx)
}
