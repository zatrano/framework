package core

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/zatrano/framework/pkg/auth"
	"github.com/zatrano/framework/pkg/cache"
	"github.com/zatrano/framework/pkg/config"
	"github.com/zatrano/framework/pkg/i18n"
	"github.com/zatrano/framework/pkg/queue"
)

// App is the root application container.
type App struct {
	Config *config.Config
	Log    *zap.Logger
	Fiber  *fiber.App

	DB    *gorm.DB
	Redis *redis.Client

	// SessionStore is set when Redis-backed sessions are enabled (use session.FromContext in handlers).
	SessionStore *session.Store

	// I18n is loaded when config i18n.enabled (JSON catalogs under locales_dir).
	I18n *i18n.Bundle

	// Gate is the resource-based authorization registry (define/check abilities).
	Gate *auth.Gate

	// RBAC is the role-based access control manager (role → permission mapping, DB-backed).
	RBAC *auth.RBACManager

	// Cache is the application cache manager (memory or Redis driver).
	Cache *cache.Manager

	// Queue is the background job queue manager (Redis-backed).
	Queue *queue.Manager
}
