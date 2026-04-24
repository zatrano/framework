package repositories

import (
	"context"

	"github.com/zatrano/framework/configs/databaseconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/requests"

	"gorm.io/gorm"
)

type ICountryRepository interface {
	GetAllCountries(ctx context.Context, params requests.CountryListParams) ([]models.Country, int64, error)
	GetCountryByID(ctx context.Context, id uint) (*models.Country, error)
	CreateCountry(ctx context.Context, country *models.Country) error
	UpdateCountry(ctx context.Context, id uint, data map[string]interface{}) error
	DeleteCountry(ctx context.Context, id uint) error
	GetCountryCount(ctx context.Context) (int64, error)
}

type CountryRepository struct {
	base IBaseRepository[models.Country]
	db   *gorm.DB
}

func NewCountryRepository() ICountryRepository {
	base := NewBaseRepository[models.Country](databaseconfig.GetDB())
	base.SetAllowedSortColumns([]string{"id", "name", "created_at"})
	return &CountryRepository{base: base, db: databaseconfig.GetDB()}
}

func (r *CountryRepository) GetAllCountries(ctx context.Context, params requests.CountryListParams) ([]models.Country, int64, error) {
	var countries []models.Country
	var totalCount int64
	query := r.db.WithContext(ctx).Model(&models.Country{})
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
		return []models.Country{}, 0, nil
	}
	query = query.Order(params.SortBy + " " + params.OrderBy).Limit(params.PerPage).Offset(params.CalculateOffset())
	if err := query.Find(&countries).Error; err != nil {
		return nil, 0, err
	}
	return countries, totalCount, nil
}

func (r *CountryRepository) GetCountryByID(ctx context.Context, id uint) (*models.Country, error) {
	return r.base.GetByID(ctx, id)
}

func (r *CountryRepository) CreateCountry(ctx context.Context, country *models.Country) error {
	return r.base.Create(ctx, country)
}

func (r *CountryRepository) UpdateCountry(ctx context.Context, id uint, data map[string]interface{}) error {
	return r.base.UpdateFields(ctx, id, data)
}

func (r *CountryRepository) DeleteCountry(ctx context.Context, id uint) error {
	return r.base.Delete(ctx, id)
}

func (r *CountryRepository) GetCountryCount(ctx context.Context) (int64, error) {
	return r.base.GetCount(ctx)
}
