package seeders

import (
	"time"

	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/models"

	"gorm.io/gorm"
)

func SeedUserTypes(db *gorm.DB) error {
	userTypes := []models.UserType{
		{Name: "Admin"},
		{Name: "User"},
	}

	logconfig.SLog.Info("Kullanıcı tipleri yükleniyor...")

	for _, userType := range userTypes {
		userType.BaseModel = models.BaseModel{
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: 1,
			UpdatedBy: 1,
		}

		if err := db.Create(&userType).Error; err != nil {
			logconfig.SLog.Error("Kullanıcı tipi eklenirken hata: "+userType.Name, err)
			return err
		}
	}

	logconfig.SLog.Info("Kullanıcı tipleri yükleme tamamlandı.")
	return nil
}
