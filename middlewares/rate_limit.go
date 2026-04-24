package middlewares

import (
	"slices"
	"time"

	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/configs/redisconfig"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	redisstorage "github.com/gofiber/storage/redis/v3"
)

var whitelistIPs = []string{"127.0.0.1", "::1"}

func shouldSkipLimit(c fiber.Ctx) bool {
	if slices.Contains(whitelistIPs, c.IP()) {
		return true
	}
	if !envconfig.IsProd() {
		return true
	}
	return false
}

// newRedisStorage — rate limiter için Redis storage oluşturur.
// Multi-instance deployment'ta sayaçlar paylaşılır.
func newRedisStorage() *redisstorage.Storage {
	return redisstorage.NewFromConnection(redisconfig.GetClient())
}

// GlobalRateLimit — tüm uygulama için IP başına genel limit.
// Redis-backed: çok sunucu kurulumunda sayaçlar paylaşılır.
func GlobalRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        envconfig.Int("GLOBAL_RATE_MAX", 300),
		Expiration: time.Minute,
		Storage:    newRedisStorage(), // ← Redis-backed (eskiden in-memory)
		KeyGenerator: func(c fiber.Ctx) string {
			return "rate:global:" + c.IP()
		},
		Next: func(c fiber.Ctx) bool {
			p := c.Path()
			if p == "/healthz" || p == "/readyz" || p == "/health" || p == "/metrics" {
				return true
			}
			return shouldSkipLimit(c)
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).
				SendString("Çok fazla istek gönderildi. Lütfen kısa bir süre sonra tekrar deneyin.")
		},
	})
}

// FormPostRateLimit — form POST'ları için IP + path başına limit.
func FormPostRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        envconfig.Int("FORM_POST_RATE_MAX", 30),
		Expiration: time.Minute,
		Storage:    newRedisStorage(),
		KeyGenerator: func(c fiber.Ctx) string {
			return "rate:form:" + c.IP() + ":" + c.Path()
		},
		Next: func(c fiber.Ctx) bool {
			if c.Method() != fiber.MethodPost {
				return true
			}
			return shouldSkipLimit(c)
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).
				SendString("Çok fazla form isteği gönderildi. Lütfen biraz bekleyin.")
		},
	})
}

// LoginRateLimit — kaba kuvvet engelleme; giriş uç noktasına özel.
func LoginRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        envconfig.Int("LOGIN_RATE_MAX", 5),
		Expiration: time.Minute,
		Storage:    newRedisStorage(),
		KeyGenerator: func(c fiber.Ctx) string {
			return "rate:login:" + c.IP()
		},
		Next: func(c fiber.Ctx) bool {
			if !(c.Method() == fiber.MethodPost && c.Path() == "/auth/login") {
				return true
			}
			return shouldSkipLimit(c)
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).
				SendString("Çok fazla giriş denemesi yaptınız. Lütfen 1 dakika sonra tekrar deneyin.")
		},
	})
}

// APIRateLimit — REST API uç noktaları için token tabanlı limit.
func APIRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        envconfig.Int("API_RATE_MAX", 120),
		Expiration: time.Minute,
		Storage:    newRedisStorage(),
		KeyGenerator: func(c fiber.Ctx) string {
			// API anahtarı varsa ona göre, yoksa IP'ye göre
			if apiKey := c.Get("X-API-Key"); apiKey != "" {
				return "rate:api:key:" + apiKey
			}
			return "rate:api:ip:" + c.IP()
		},
		Next: func(c fiber.Ctx) bool {
			return shouldSkipLimit(c)
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fiber.Map{
					"status":  429,
					"code":    "RATE_LIMITED",
					"message": "API rate limit aşıldı. Lütfen bekleyin.",
				},
			})
		},
	})
}
