package database

import (
	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/database/migrations"
	"github.com/zatrano/framework/database/seeders"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Initialize(db *gorm.DB, migrate bool, seed bool) {
	if !migrate && !seed {
		logconfig.SLog.Info("Migrate veya seed bayrağı belirtilmedi, işlem yapılmayacak.")
		return
	}

	tx := db.Begin()
	if tx.Error != nil {
		logconfig.Log.Fatal("Veritabanı işlemine başlanamadı", zap.Error(tx.Error))
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			logconfig.Log.Fatal("Veritabanı başlatma işlemi başarısız oldu, panic ile geri alındı", zap.Any("panic_info", r))
		}
	}()

	logconfig.SLog.Info("Veritabanı başlatma işlemi başlıyor...")

	// --- Migrasyon ---
	if migrate {
		logconfig.SLog.Info("Migrasyonlar çalıştırılıyor...")
		if err := migrations.MigrateAll(tx); err != nil {
			_ = tx.Rollback()
			logconfig.Log.Fatal("Migrasyon işlemi başarısız oldu", zap.Error(err))
		}
		logconfig.SLog.Info("Migrasyon işlemi tamamlandı.")
	} else {
		logconfig.SLog.Info("Migrate bayrağı belirtilmedi, migrasyon adımı atlanıyor.")
	}

	// --- Seed (veri oluşturma) ---
	if seed {
		logconfig.SLog.Info("Seeder işlemleri başlatılıyor...")
		if err := seeders.SeedAll(tx); err != nil {
			_ = tx.Rollback()
			logconfig.Log.Fatal("Seed işlemi başarısız oldu", zap.Error(err))
		}
		logconfig.SLog.Info("Seeder işlemleri tamamlandı.")
	} else {
		logconfig.SLog.Info("Seed bayrağı belirtilmedi, seeding adımı atlanıyor.")
	}

	// --- Commit ---
	logconfig.SLog.Info("Veritabanı işlemi commit ediliyor...")
	if err := tx.Commit().Error; err != nil {
		logconfig.Log.Fatal("Commit işlemi başarısız oldu", zap.Error(err))
	}

	logconfig.SLog.Info("Veritabanı başlatma işlemi başarıyla tamamlandı.")
}
