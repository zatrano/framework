package redisconfig

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/configs/logconfig"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

type RedisConfig struct {
	Host      string
	Port      int
	Password  string
	DB        int
	UseTLS    bool
	ResetKeys bool
}

// InitRedis — Redis bağlantısını başlatır
func InitRedis() {
	cfg := loadConfig()
	opt := buildOptions(cfg)

	RedisClient = redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RedisClient.Ping(ctx).Err(); err != nil {
		logconfig.SLog.Fatalw("Redis bağlantısı başarısız",
			"host", cfg.Host,
			"port", cfg.Port,
			"db", cfg.DB,
			"tls", cfg.UseTLS,
			"error", err,
		)
	}

	if cfg.ResetKeys {
		if err := RedisClient.FlushAll(ctx).Err(); err != nil {
			logconfig.SLog.Errorw("Redis anahtarlarını temizleme başarısız", "error", err)
		} else {
			logconfig.SLog.Warn("Redis tüm anahtarlar temizlendi (REDIS_RESET=true)")
		}
	}

	logconfig.SLog.Infow("Redis bağlantısı başarılı",
		"host", cfg.Host,
		"port", cfg.Port,
		"db", cfg.DB,
		"tls", cfg.UseTLS,
	)
}

// Config yükleme
func loadConfig() RedisConfig {
	return RedisConfig{
		Host:      envconfig.String("REDIS_HOST", "127.0.0.1"),
		Port:      envconfig.Int("REDIS_PORT", 6379),
		Password:  envconfig.String("REDIS_PASSWORD", ""),
		DB:        envconfig.Int("REDIS_DB", 0),
		UseTLS:    strings.EqualFold(envconfig.String("REDIS_TLS", "false"), "true"),
		ResetKeys: strings.EqualFold(envconfig.String("REDIS_RESET", "false"), "true"),
	}
}

// Redis.Options oluştur
func buildOptions(cfg RedisConfig) *redis.Options {
	opt := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     50,
		MinIdleConns: 5,
	}

	if cfg.UseTLS && envconfig.IsProd() {
		opt.TLSConfig = &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		}
	}

	return opt
}

// GetClient — Redis istemcisini döner
func GetClient() *redis.Client {
	if RedisClient == nil {
		InitRedis()
	}
	return RedisClient
}

// Helper: key prefix ekle
func GetPrefixedKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", prefix, key)
}

// Close Redis bağlantısını kapatır. main'de defer redisconfig.Close() ile çağrılmalıdır.
func Close() error {
	if RedisClient == nil {
		return nil
	}
	err := RedisClient.Close()
	if err != nil {
		logconfig.SLog.Errorw("Redis bağlantısı kapatılırken hata oluştu", "error", err)
		return err
	}
	RedisClient = nil
	logconfig.SLog.Info("Redis bağlantısı kapatıldı.")
	return nil
}
