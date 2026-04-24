package repositories

import (
	"context"

	"github.com/zatrano/framework/configs/databaseconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/requests"

	"gorm.io/gorm"
)

type IUserTypeRepository interface {
	GetAllUserTypes(ctx context.Context, params requests.UserTypeListParams) ([]models.UserType, int64, error)
	GetUserTypeByID(ctx context.Context, id uint) (*models.UserType, error)
	CreateUserType(ctx context.Context, u *models.UserType) error
	UpdateUserType(ctx context.Context, id uint, data map[string]interface{}) error
	DeleteUserType(ctx context.Context, id uint) error
}

type UserTypeRepository struct {
	base IBaseRepository[models.UserType]
	db   *gorm.DB
}

func NewUserTypeRepository() IUserTypeRepository {
	base := NewBaseRepository[models.UserType](databaseconfig.GetDB())
	base.SetAllowedSortColumns([]string{"id", "name"})
	return &UserTypeRepository{base: base, db: databaseconfig.GetDB()}
}

func (r *UserTypeRepository) GetAllUserTypes(ctx context.Context, params requests.UserTypeListParams) ([]models.UserType, int64, error) {
	var userTypes []models.UserType
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&models.UserType{})

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
		return []models.UserType{}, 0, nil
	}

	query = query.Order(params.SortBy + " " + params.OrderBy)

	offset := params.CalculateOffset()
	query = query.Limit(params.PerPage).Offset(offset)

	if err := query.Find(&userTypes).Error; err != nil {
		return nil, 0, err
	}

	return userTypes, totalCount, nil
}

func (r *UserTypeRepository) GetUserTypeByID(ctx context.Context, id uint) (*models.UserType, error) {
	return r.base.GetByID(ctx, id)
}

func (r *UserTypeRepository) CreateUserType(ctx context.Context, u *models.UserType) error {
	return r.base.Create(ctx, u)
}

func (r *UserTypeRepository) UpdateUserType(ctx context.Context, id uint, data map[string]interface{}) error {
	return r.base.UpdateFields(ctx, id, data)
}

func (r *UserTypeRepository) DeleteUserType(ctx context.Context, id uint) error {
	return r.base.Delete(ctx, id)
}
