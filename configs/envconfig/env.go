package envconfig

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// LoadIfDev, APP_ENV != "production" ise .env dosyasını yükler.
func LoadIfDev() {
	if os.Getenv("APP_ENV") != "production" {
		_ = godotenv.Load()
	}
}

// Load, çalışma dizininden .env dosyasını yükler (mevcutsa). Mevcut env değişkenlerini ezmez.
// Production'da GOOGLE_REDIRECT_URI vb. .env'den okunabilsin diye çağrılabilir.
func Load() {
	_ = godotenv.Load()
}

// IsProd, APP_ENV == "production" ise true döner.
func IsProd() bool {
	return os.Getenv("APP_ENV") == "production"
}

// String, çevresel değişken değerini döner; boşsa defaultValue kullanılır.
func String(key string, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// Int, çevresel değişken değerini int olarak döner; boş veya geçersizse defaultValue döner.
func Int(key string, defaultValue int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}
	return defaultValue
}

// Float, çevresel değişken değerini float64 olarak döner; boş veya geçersizse defaultValue döner.
func Float(key string, defaultValue float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	return defaultValue
}
