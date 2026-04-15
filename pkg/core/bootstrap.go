package core

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/zatrano/framework/pkg/auth"
	"github.com/zatrano/framework/pkg/cache"
	"github.com/zatrano/framework/pkg/config"
	"github.com/zatrano/framework/pkg/i18n"
	"github.com/zatrano/framework/pkg/validation"
)

// Bootstrap builds logger and optional database/redis clients from configuration.
func Bootstrap(cfg *config.Config) (*App, error) {
	zl, err := newZapLogger(cfg)
	if err != nil {
		return nil, err
	}

	app := &App{
		Config: cfg,
		Log:    zl,
	}

	if u := strings.TrimSpace(cfg.DatabaseURL); u != "" {
		gormLog := logger.Default.LogMode(logger.Warn)
		if cfg.LogDevelopment {
			gormLog = logger.New(
				zap.NewStdLog(zl),
				logger.Config{
					SlowThreshold:             200 * time.Millisecond,
					LogLevel:                  logger.Info,
					IgnoreRecordNotFoundError: true,
					Colorful:                  true,
				},
			)
		}
		db, err := gorm.Open(postgres.Open(u), &gorm.Config{Logger: gormLog})
		if err != nil {
			return nil, fmt.Errorf("postgres: %w", err)
		}
		app.DB = db
	}

	if u := strings.TrimSpace(cfg.RedisURL); u != "" {
		opt, err := redis.ParseURL(u)
		if err != nil {
			return nil, fmt.Errorf("redis url: %w", err)
		}
		app.Redis = redis.NewClient(opt)
	}

	if cfg.I18n.Enabled {
		bundle, err := i18n.LoadDir(cfg.I18n.LocalesDir, cfg.I18n.DefaultLocale, cfg.I18n.SupportedLocales)
		if err != nil {
			return nil, fmt.Errorf("i18n: %w", err)
		}
		app.I18n = bundle
	}

	// Initialise the validation engine (i18n bundle may be nil — engine handles it).
	validation.Init(app.I18n)

	// Gate is always available (resource-based authorization).
	app.Gate = auth.NewGate()

	// RBAC requires DB — initialise when available, log and continue if cache warm fails.
	if app.DB != nil {
		rbac, err := auth.NewRBACManager(app.DB)
		if err != nil {
			zl.Warn("rbac: cache warm failed (tables may not exist yet, run migrations)", zap.Error(err))
		} else {
			app.RBAC = rbac
		}
	}

	// Initialise Cache. Redis is preferred if available.
	if app.Redis != nil {
		app.Cache = cache.New(cache.NewRedisDriver(app.Redis))
	} else {
		app.Cache = cache.New(cache.NewMemoryDriver())
	}

	return app, nil
}

func newZapLogger(cfg *config.Config) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(strings.ToLower(cfg.LogLevel))
	if err != nil {
		return nil, err
	}
	zcfg := zap.NewProductionConfig()
	if cfg.LogDevelopment {
		zcfg = zap.NewDevelopmentConfig()
	}
	zcfg.Level = zap.NewAtomicLevelAt(level)
	return zcfg.Build()
}

// Close releases resources (database/sql, redis).
func (a *App) Close() error {
	var errs []error
	if a.DB != nil {
		sqlDB, err := a.DB.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, fmt.Errorf("close sql: %w", err))
			}
		} else {
			errs = append(errs, err)
		}
	}
	if a.Redis != nil {
		if err := a.Redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close redis: %w", err))
		}
	}
	return errors.Join(errs...)
}

// NewFiber creates the Fiber application with framework defaults (timeouts, error handling).
func NewFiber(a *App) *fiber.App {
	cfg := fiber.Config{
		AppName:      a.Config.AppName,
		ServerHeader: "ZATRANO",
		ReadTimeout:  a.Config.HTTPReadTimeout,
		ErrorHandler: a.errorHandler,
	}
	if n := a.Config.HTTP.BodyLimit; n > 0 {
		cfg.BodyLimit = n
	}
	return fiber.New(cfg)
}

func (a *App) errorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	fields := []zap.Field{
		zap.String("method", c.Method()),
		zap.String("path", c.Path()),
		zap.Int("status", code),
		zap.Error(err),
	}
	if rid := requestid.FromContext(c); rid != "" {
		fields = append(fields, zap.String("request_id", rid))
	}
	a.Log.Warn("http error", fields...)

	errBody := fiber.Map{
		"code":    code,
		"message": err.Error(),
	}
	if rid := requestid.FromContext(c); rid != "" {
		errBody["request_id"] = rid
	}
	return c.Status(code).JSON(fiber.Map{"error": errBody})
}
