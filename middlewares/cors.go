package middlewares

import (
	"strings"

	"github.com/zatrano/framework/configs/envconfig"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

// CORS — production'da COOKIE_DOMAIN'e göre konfigüre edilir.
// Development'ta tüm origin'lere izin verir.
func CORS() fiber.Handler {
	allowedOrigins := envconfig.String("CORS_ALLOWED_ORIGINS", "")

	if !envconfig.IsProd() || allowedOrigins == "" {
		// Development: tüm origin
		return cors.New(cors.Config{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "X-CSRF-Token"},
			ExposeHeaders:    []string{"X-Request-ID"},
			AllowCredentials: false,
		})
	}

	// Production: izin verilen origin'ler virgülle ayrılmış
	origins := strings.Split(allowedOrigins, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}

	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "X-CSRF-Token"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           86400, // preflight cache: 24 saat
	})
}
