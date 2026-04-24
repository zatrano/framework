package migrations

import (
	"fmt"
	"strings"

	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MigrateAll veritabanı şemasını tek adımda GORM AutoMigrate ile oluşturur veya günceller.
// Yeni model eklendiğinde yalnızca bu dosyadaki sıraya (FK bağımlılığına uygun) eklenir.
func MigrateAll(db *gorm.DB) error {
	logconfig.SLog.Info("Veritabanı şeması migrate ediliyor...")

	entities := []interface{}{
		&models.Definition{},
		&models.UserType{},
		&models.User{},
		&models.Country{},
		&models.City{},
		&models.District{},
		&models.Address{},
		&models.ContactMessage{},
	}

	for _, entity := range entities {
		name := entityTypeName(entity)
		if err := db.AutoMigrate(entity); err != nil {
			logconfig.Log.Error("AutoMigrate başarısız", zap.String("model", name), zap.Error(err))
			return fmt.Errorf("migrate %s: %w", name, err)
		}
		logconfig.SLog.Info("Tablo hazır: " + name)
	}

	logconfig.SLog.Info("Veritabanı şeması migrate tamamlandı.")
	return nil
}

func entityTypeName(m interface{}) string {
	t := fmt.Sprintf("%T", m)
	parts := strings.Split(t, ".")
	if len(parts) == 0 {
		return t
	}
	return parts[len(parts)-1]
}
