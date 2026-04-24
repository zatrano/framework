package seeders

import (
	"time"

	"github.com/zatrano/framework/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func SeedDefinitions(db *gorm.DB) error {
	now := time.Now()
	items := []models.Definition{
		{
			Key:         "app_name",
			Value:       "github.com/zatrano/framework",
			Description: "Uygulama adı",
			BaseModel: models.BaseModel{
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: 1,
				UpdatedBy: 1,
			},
		},
		// website/contact: sol sütun iletişim + harita
		{
			Key:         "contact_address",
			Value:       "Örnek Mahalle, 34000 İstanbul, Türkiye",
			Description: "İletişim sayfası: adres",
			BaseModel: models.BaseModel{
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: 1,
				UpdatedBy: 1,
			},
		},
		{
			Key:         "contact_email",
			Value:       "info@example.com",
			Description: "İletişim sayfası: e-posta",
			BaseModel: models.BaseModel{
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: 1,
				UpdatedBy: 1,
			},
		},
		{
			Key:         "contact_phone",
			Value:       "+90 (212) 000 00 00",
			Description: "İletişim sayfası: telefon",
			BaseModel: models.BaseModel{
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: 1,
				UpdatedBy: 1,
			},
		},
		{
			Key:   "contact_map_embed_url",
			Value: "",
			// Harita opsiyonel; değer verilirse iframe src olur (Google Maps embed URL)
			Description: "İletişim sayfası: harita embed URL",
			BaseModel: models.BaseModel{
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
				CreatedBy: 1,
				UpdatedBy: 1,
			},
		},
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "description", "updated_at", "updated_by", "is_active"}),
	}).Create(&items).Error
}
