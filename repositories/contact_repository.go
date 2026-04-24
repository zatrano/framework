package repositories

import (
	"context"
	"time"

	"github.com/zatrano/framework/configs/databaseconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/requests"

	"gorm.io/gorm"
)

type IContactRepository interface {
	Create(ctx context.Context, msg *models.ContactMessage) error
	ListPaginated(ctx context.Context, params requests.ContactMessageListParams) ([]models.ContactMessage, int64, error)
	GetByID(ctx context.Context, id uint) (*models.ContactMessage, error)
	MarkRead(ctx context.Context, id uint) error
}

type ContactRepository struct {
	db *gorm.DB
}

func NewContactRepository() IContactRepository {
	return &ContactRepository{db: databaseconfig.GetDB()}
}

func (r *ContactRepository) Create(ctx context.Context, msg *models.ContactMessage) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *ContactRepository) ListPaginated(ctx context.Context, params requests.ContactMessageListParams) ([]models.ContactMessage, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.ContactMessage{})

	if params.Name != "" {
		q = q.Where("name ILIKE ?", "%"+params.Name+"%")
	}
	if params.Email != "" {
		q = q.Where("email ILIKE ?", "%"+params.Email+"%")
	}
	if params.Subject != "" {
		q = q.Where("subject ILIKE ?", "%"+params.Subject+"%")
	}
	if params.Unread == "1" {
		q = q.Where("read_at IS NULL")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []models.ContactMessage{}, 0, nil
	}

	orderBy := params.SortBy + " " + params.OrderBy
	q = q.Order(orderBy)

	offset := params.CalculateOffset()
	q = q.Offset(offset).Limit(params.PerPage)

	var list []models.ContactMessage
	if err := q.Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *ContactRepository) GetByID(ctx context.Context, id uint) (*models.ContactMessage, error) {
	var m models.ContactMessage
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ContactRepository) MarkRead(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.ContactMessage{}).Where("id = ?", id).Update("read_at", now).Error
}
