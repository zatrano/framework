package middlewares

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
)

// Compress — gzip/deflate/brotli sıkıştırma.
// Statik dosyalar, HTML, JSON ve API yanıtları sıkıştırılır.
// /uploads ve /metrics uç noktaları atlanır.
func Compress() fiber.Handler {
	return compress.New(compress.Config{
		Level: compress.LevelBestSpeed, // hız/boyut dengesi
		Next: func(c fiber.Ctx) bool {
			// Küçük yanıtlar ve belirli path'ler atlanır
			path := c.Path()
			if path == "/metrics" || path == "/health" {
				return true // sıkıştırma yapma
			}
			return false
		},
	})
}
