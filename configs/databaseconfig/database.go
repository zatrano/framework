package databaseconfig

import (
	"context"
	"strconv"
	"time"

	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/models"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
	TimeZone string
}

func InitDB() {
	// .env yükleme işi main.go'da dev modda yapılır; burada tekrar etmiyoruz.

	dbConfig := DatabaseConfig{
		Host:     envconfig.String("DB_HOST", "localhost"),
		Port:     envconfig.Int("DB_PORT", 5432),
		User:     envconfig.String("DB_USERNAME", "postgres"),
		Password: envconfig.String("DB_PASSWORD", ""),
		Name:     envconfig.String("DB_DATABASE", "github.com/zatrano/framework"),
		SSLMode:  envconfig.String("DB_SSL_MODE", "disable"),
		TimeZone: envconfig.String("DB_TIMEZONE", "UTC"),
	}

	logconfig.Log.Info("Database configuration loaded",
		zap.String("host", dbConfig.Host),
		zap.Int("port", dbConfig.Port),
		zap.String("user", dbConfig.User),
		zap.String("database", dbConfig.Name),
		zap.String("sslmode", dbConfig.SSLMode),
		zap.String("timezone", dbConfig.TimeZone),
	)

	dsn := "host=" + dbConfig.Host +
		" user=" + dbConfig.User +
		" password=" + dbConfig.Password +
		" dbname=" + dbConfig.Name +
		" port=" + strconv.Itoa(dbConfig.Port) +
		" sslmode=" + dbConfig.SSLMode +
		" TimeZone=" + dbConfig.TimeZone

	var gormerr error
	DB, gormerr = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:  logger.Default.LogMode(getGormLogLevel()),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if gormerr != nil {
		logconfig.Log.Fatal("Failed to connect to database",
			zap.String("host", dbConfig.Host),
			zap.Int("port", dbConfig.Port),
			zap.String("user", dbConfig.User),
			zap.String("database", dbConfig.Name),
			zap.Error(gormerr),
		)
	}

	models.RegisterBaseModelCallbacks(DB)

	sqlDB, err := DB.DB()
	if err != nil {
		logconfig.Log.Fatal("Failed to get underlying sql.DB instance", zap.Error(err))
	}

	maxIdleConns := envconfig.Int("DB_MAX_IDLE_CONNS", 10)
	maxOpenConns := envconfig.Int("DB_MAX_OPEN_CONNS", 100)
	connMaxLifetimeMinutes := envconfig.Int("DB_CONN_MAX_LIFETIME_MINUTES", 60)

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetimeMinutes) * time.Minute)

	logconfig.Log.Info("Database connection established successfully",
		zap.Int("max_idle_conns", maxIdleConns),
		zap.Int("max_open_conns", maxOpenConns),
		zap.Int("conn_max_lifetime_minutes", connMaxLifetimeMinutes),
	)
}

// Health/Readiness ping
func Ping() error {
	if DB == nil {
		return gorm.ErrInvalidDB
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	return sqlDB.PingContext(ctx)
}

func getGormLogLevel() logger.LogLevel {
	switch envconfig.String("DB_LOG_LEVEL", "info") {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	default:
		return logger.Info
	}
}

func GetDB() *gorm.DB {
	if DB == nil {
		logconfig.Log.Fatal("Database connection not initialized. Call InitDB() first.")
	}
	return DB
}

func CloseDB() error {
	if DB == nil {
		logconfig.SLog.Info("Database connection already closed or not initialized.")
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		logconfig.Log.Error("Failed to get database instance for closing", zap.Error(err))
		return err
	}

	err = sqlDB.Close()
	if err != nil {
		logconfig.Log.Error("Error closing database connection", zap.Error(err))
		return err
	}

	logconfig.SLog.Info("Database connection closed successfully.")
	DB = nil
	return nil
}


