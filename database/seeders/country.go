package seeders

import (
	"time"

	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/models"

	"gorm.io/gorm"
)

func SeedCountries(db *gorm.DB) error {
	logconfig.SLog.Info("Ülkeler yükleniyor...")

	countries := []models.Country{
		{
			Name: "Türkiye",
		},
	}

	for _, country := range countries {
		country.BaseModel = models.BaseModel{
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: 1,
			UpdatedBy: 1,
		}

		if err := db.Create(&country).Error; err != nil {
			logconfig.SLog.Error("Ülke eklenirken hata: "+country.Name, err)
			return err
		}

	}

	logconfig.SLog.Info("Ülke yükleme tamamlandı.")
	return nil
}
