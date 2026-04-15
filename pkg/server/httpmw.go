package server

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/fiber/v3/middleware/timeout"
	redisstorage "github.com/gofiber/storage/redis/v3"

	"github.com/zatrano/framework/pkg/core"
)

// registerHTTPMiddleware installs CORS, per-request timeout, and rate limiting from config.
// Call after recover + requestid, before helmet (so preflight sees CORS early).
func registerHTTPMiddleware(a *core.App, app *fiber.App) {
	h := a.Config.HTTP

	if h.CORSEnabled {
		cc := cors.Config{
			AllowOrigins:     h.CORSAllowOrigins,
			AllowMethods:     h.CORSAllowMethods,
			AllowHeaders:     h.CORSAllowHeaders,
			ExposeHeaders:    h.CORSExposeHeaders,
			AllowCredentials: h.CORSAllowCredentials,
			MaxAge:           h.CORSMaxAge,
		}
		app.Use(cors.New(cc))
	}

	if h.RequestTimeout > 0 {
		app.Use(timeout.New(func(c fiber.Ctx) error { return c.Next() }, timeout.Config{
			Timeout: h.RequestTimeout,
			OnTimeout: func(c fiber.Ctx) error {
				errObj := fiber.Map{
					"code":    fiber.StatusRequestTimeout,
					"message": "request timeout",
				}
				if rid := requestid.FromContext(c); rid != "" {
					errObj["request_id"] = rid
				}
				return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{"error": errObj})
			},
		}))
	}

	if h.RateLimitEnabled {
		lc := limiter.Config{
			Max:        h.RateLimitMax,
			Expiration: h.RateLimitWindow,
			LimitReached: func(c fiber.Ctx) error {
				errObj := fiber.Map{
					"code":    fiber.StatusTooManyRequests,
					"message": "too many requests",
				}
				if rid := requestid.FromContext(c); rid != "" {
					errObj["request_id"] = rid
				}
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": errObj})
			},
		}
		if h.RateLimitRedis && a.Redis != nil {
			lc.Storage = redisstorage.NewFromConnection(a.Redis)
		}
		app.Use(limiter.New(lc))
	}
}
