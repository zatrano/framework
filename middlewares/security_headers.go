package middlewares

import (
	"strings"

	"github.com/zatrano/framework/configs/envconfig"

	"github.com/gofiber/fiber/v3"
)

func SecurityHeaders() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "SAMEORIGIN")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		if envconfig.IsProd() {
			baseURL := envconfig.String("APP_BASE_URL", "")
			if strings.HasPrefix(baseURL, "https://") {
				c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}
		}
		return c.Next()
	}
}
