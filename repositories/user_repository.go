package repositories

import (
	"context"

	"github.com/zatrano/framework/configs/databaseconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/requests"

	"gorm.io/gorm"
)

type IUserRepository interface {
	GetAllUsers(ctx context.Context, params requests.UserListParams) ([]models.User, int64, error)
	GetUserByID(ctx context.Context, id uint) (*models.User, error)
	CreateUser(ctx context.Context, u *models.User) error
	UpdateUser(ctx context.Context, id uint, data map[string]interface{}) error
	DeleteUser(ctx context.Context, id uint) error
	GetUserCount(ctx context.Context) (int64, error)
}

type UserRepository struct {
	base IBaseRepository[models.User]
	db   *gorm.DB
}

func NewUserRepository() IUserRepository {
	base := NewBaseRepository[models.User](databaseconfig.GetDB())
	base.SetAllowedSortColumns([]string{"id", "name", "email", "created_at"})
	return &UserRepository{base: base, db: databaseconfig.GetDB()}
}

func (r *UserRepository) GetAllUsers(ctx context.Context, params requests.UserListParams) ([]models.User, int64, error) {
	var users []models.User
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&models.User{})

	if params.Name != "" {
		query = query.Where("name ILIKE ?", "%"+params.Name+"%")
	}

	if params.Email != "" {
		query = query.Where("email ILIKE ?", "%"+params.Email+"%")
	}

	if params.IsActive != "" {
		switch params.IsActive {
		case "true":
			query = query.Where("is_active = ?", true)
		case "false":
			query = query.Where("is_active = ?", false)
		}
	}

	if params.EmailVerified != "" {
		switch params.EmailVerified {
		case "true":
			query = query.Where("email_verified = ?", true)
		case "false":
			query = query.Where("email_verified = ?", false)
		}
	}

	if params.UserTypeID != nil {
		query = query.Where("user_type_id = ?", *params.UserTypeID)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	if totalCount == 0 {
		return []models.User{}, 0, nil
	}

	query = query.Order(params.SortBy + " " + params.OrderBy)

	offset := params.CalculateOffset()
	query = query.Limit(params.PerPage).Offset(offset)

	query = query.Preload("UserType")

	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, totalCount, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Preload("UserType").
		First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, u *models.User) error {
	return r.base.Create(ctx, u)
}

func (r *UserRepository) UpdateUser(ctx context.Context, id uint, data map[string]interface{}) error {
	return r.base.UpdateFields(ctx, id, data)
}

func (r *UserRepository) DeleteUser(ctx context.Context, id uint) error {
	return r.base.Delete(ctx, id)
}

func (r *UserRepository) GetUserCount(ctx context.Context) (int64, error) {
	return r.base.GetCount(ctx)
}
