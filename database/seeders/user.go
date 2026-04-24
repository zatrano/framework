package seeders

import (
	"time"

	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SeedSystemUser(db *gorm.DB) error {
	adminUserType := uint(1)

	admin := models.User{
		Name:       "github.com/zatrano/framework",
		Email:      "github.com/zatrano/framework@zatrano.com",
		UserTypeID: adminUserType,
		BaseModel: models.BaseModel{
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: 1,
			UpdatedBy: 1,
		},
		EmailVerified: true,
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("github.com/zatrano/framework"), bcrypt.DefaultCost)
	if err != nil {
		logconfig.SLog.Error("Sistem kullanıcısının şifresi hash'lenirken hata oluştu", err)
		return err
	}
	admin.Password = string(hashedPassword)

	logconfig.SLog.Info("Sistem kullanıcısı oluşturuluyor...")

	if err := db.Create(&admin).Error; err != nil {
		logconfig.SLog.Error("Sistem kullanıcısı oluşturulamadı: "+admin.Email, err)
		return err
	}

	logconfig.SLog.Info("Sistem kullanıcısı başarıyla oluşturuldu: " + admin.Email)
	return nil
}
