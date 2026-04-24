package main

import (
	"flag"
	"log"

	"github.com/zatrano/framework/configs/databaseconfig"
	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/database"
)

func main() {
	// Uygulama ile aynı davranış: .env her ortamda okunur, dev için geriye dönük çağrı korunur.
	envconfig.Load()
	envconfig.LoadIfDev()

	// Günlükleyici başlat
	logconfig.InitLogger()
	defer logconfig.SyncLogger()

	// Komut satırı parametreleri
	migrateFlag := flag.Bool("migrate", false, "Veritabanı migrasyonlarını çalıştır")
	seedFlag := flag.Bool("seed", false, "Veritabanı seederlarını çalıştır")
	flag.Parse()

	// DB başlat
	databaseconfig.InitDB()
	defer func() {
		if err := databaseconfig.CloseDB(); err != nil {
			log.Println("Database kapanırken hata:", err)
		}
	}()

	db := databaseconfig.GetDB()

	logconfig.SLog.Infow("Veritabanı başlatma işlemi çalıştırılıyor",
		"migrate", *migrateFlag,
		"seed", *seedFlag,
	)

	// Migrasyon ve seed işlemleri
	database.Initialize(db, *migrateFlag, *seedFlag)

	logconfig.SLog.Info("Veritabanı başlatma işlemi tamamlandı.")
}
