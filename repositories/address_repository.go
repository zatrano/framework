package repositories

import (
	"context"

	"github.com/zatrano/framework/configs/databaseconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/requests"

	"gorm.io/gorm"
)

type IAddressRepository interface {
	GetAllAddresses(ctx context.Context, params requests.AddressListParams) ([]models.Address, int64, error)
	GetAddressByID(ctx context.Context, id uint) (*models.Address, error)
	CreateAddress(ctx context.Context, m *models.Address) error
	UpdateAddress(ctx context.Context, id uint, data map[string]interface{}) error
	DeleteAddress(ctx context.Context, id uint) error
	GetAddressCount(ctx context.Context) (int64, error)
}

type AddressRepository struct {
	base IBaseRepository[models.Address]
	db   *gorm.DB
}

func NewAddressRepository() IAddressRepository {
	base := NewBaseRepository[models.Address](databaseconfig.GetDB())
	base.SetAllowedSortColumns([]string{"id", "detail", "created_at"})
	return &AddressRepository{base: base, db: databaseconfig.GetDB()}
}

func (r *AddressRepository) GetAllAddresses(ctx context.Context, params requests.AddressListParams) ([]models.Address, int64, error) {
	var list []models.Address
	var total int64
	query := r.db.WithContext(ctx).Model(&models.Address{}).
		Preload("Country").Preload("City").Preload("District")
	if params.CountryID != nil {
		query = query.Where("country_id = ?", *params.CountryID)
	}
	if params.CityID != nil {
		query = query.Where("city_id = ?", *params.CityID)
	}
	if params.DistrictID != nil {
		query = query.Where("district_id = ?", *params.DistrictID)
	}
	if params.Detail != "" {
		query = query.Where("detail ILIKE ?", "%"+params.Detail+"%")
	}
	if params.IsActive != "" {
		switch params.IsActive {
		case "true":
			query = query.Where("is_active = ?", true)
		case "false":
			query = query.Where("is_active = ?", false)
		}
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []models.Address{}, 0, nil
	}
	query = query.Order(params.SortBy + " " + params.OrderBy).Limit(params.PerPage).Offset(params.CalculateOffset())
	if err := query.Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *AddressRepository) GetAddressByID(ctx context.Context, id uint) (*models.Address, error) {
	var m models.Address
	err := r.db.WithContext(ctx).Preload("Country").Preload("City").Preload("District").First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *AddressRepository) CreateAddress(ctx context.Context, m *models.Address) error {
	return r.base.Create(ctx, m)
}

func (r *AddressRepository) UpdateAddress(ctx context.Context, id uint, data map[string]interface{}) error {
	return r.base.UpdateFields(ctx, id, data)
}

func (r *AddressRepository) DeleteAddress(ctx context.Context, id uint) error {
	return r.base.Delete(ctx, id)
}

func (r *AddressRepository) GetAddressCount(ctx context.Context) (int64, error) {
	return r.base.GetCount(ctx)
}
