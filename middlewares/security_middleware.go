// Package middlewares içinde ek güvenlik yardımcıları (InputSanitizer, ContentTypeEnforcer, RequestSizeLimiter).
// HTTP güvenlik başlıkları için security_headers.go içindeki SecurityHeaders kullanılır.
//
// Kayıt:
//   - InputSanitizer: main.go genel zincir (SecurityHeaders sonrası)
//   - ContentTypeEnforcer + RequestSizeLimiter: api/v1/routes.SetupAPIRoutes içinde /api/v1 grubu
package middlewares

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

// InputSanitizer tüm POST/PUT/PATCH isteklerinde form değerlerini loglar ve
// tehlikeli pattern'leri erken yakalar.
//
// NOT: Bu middleware tam sanitizasyonu DEĞİL, açık saldırı girişimlerini loglar.
// Gerçek sanitizasyon service katmanında sanitizer paketi ile yapılır.
//
// Kullanım:
//
//	app.Use(middlewares.InputSanitizer())
func InputSanitizer() fiber.Handler {
	return func(c fiber.Ctx) error {
		method := strings.ToUpper(c.Method())

		// Sadece veri gönderen metodları kontrol et
		if method == "POST" || method == "PUT" || method == "PATCH" {
			// Content-Type kontrolü
			ct := c.Get("Content-Type")

			if strings.Contains(ct, "application/x-www-form-urlencoded") ||
				strings.Contains(ct, "multipart/form-data") {
				// Form verilerini tara
				c.Request().PostArgs().VisitAll(func(key, value []byte) {
					detectAndLogSuspicious(c, string(key), string(value))
				})
			}
		}

		return c.Next()
	}
}

// suspiciousPatterns SQL Injection ve XSS için hızlı tarama desenleri.
var suspiciousPatterns = []string{
	// SQL Injection belirteçleri
	"' OR ",
	"\" OR ",
	"' AND ",
	"\" AND ",
	"'; DROP",
	"\"; DROP",
	"UNION SELECT",
	"union select",
	"UNION ALL SELECT",
	"0x", // hex encoding
	"SLEEP(",
	"sleep(",
	"BENCHMARK(",
	"benchmark(",
	"LOAD_FILE(",
	"INTO OUTFILE",
	"INFORMATION_SCHEMA",
	// XSS belirteçleri
	"<script",
	"</script",
	"javascript:",
	"vbscript:",
	"onload=",
	"onerror=",
	"onclick=",
	"onmouseover=",
	"<iframe",
	"<object",
	"<embed",
	"document.cookie",
	"document.write",
	"eval(",
	"alert(",
	"prompt(",
	"confirm(",
	// Path traversal
	"../",
	"..\\",
	"%2e%2e",
	"%2f",
}

// detectAndLogSuspicious şüpheli input'ları tespit eder ve loglar.
// Fiber'ın requestid middleware'i ile korelasyon sağlanır.
func detectAndLogSuspicious(c fiber.Ctx, key, value string) {
	upperValue := strings.ToUpper(value)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(upperValue, strings.ToUpper(pattern)) {
			// Zap logger ile logla (request_id korelasyonu ile)
			// Gerçek projede: zap.L().Warn("suspicious input detected", ...)
			// Burada fiber context'e flag bırakıyoruz
			c.Locals("security_warning", true)
			c.Locals("security_field", key)
			break
		}
	}
}

// ContentTypeEnforcer API rotalarında Content-Type zorunluluğu sağlar.
// POST/PUT/PATCH isteklerinde Content-Type header'ı yoksa 415 döner.
//
// Kullanım (sadece API group'a uygulayın):
//
//	apiGroup.Use(middlewares.ContentTypeEnforcer("application/json"))
func ContentTypeEnforcer(expectedContentType string) fiber.Handler {
	return func(c fiber.Ctx) error {
		method := strings.ToUpper(c.Method())
		if method == "POST" || method == "PUT" || method == "PATCH" {
			ct := c.Get("Content-Type")
			if !strings.Contains(ct, expectedContentType) {
				return c.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{
					"error": fiber.Map{
						"status":  415,
						"code":    "UNSUPPORTED_MEDIA_TYPE",
						"message": "Content-Type " + expectedContentType + " olmalıdır",
					},
				})
			}
		}
		return c.Next()
	}
}

// RequestSizeLimiter istek boyutunu sınırlar (büyük payload saldırılarına karşı).
// Fiber'ın BodyLimit'ini tamamlar; spesifik route'lar için kullanılır.
//
// Kullanım:
//
//	uploadGroup.Use(middlewares.RequestSizeLimiter(10 * 1024 * 1024)) // 10MB
func RequestSizeLimiter(maxBytes int64) fiber.Handler {
	return func(c fiber.Ctx) error {
		if int64(len(c.Body())) > maxBytes {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"error": fiber.Map{
					"status":  413,
					"code":    "PAYLOAD_TOO_LARGE",
					"message": "İstek boyutu çok büyük",
				},
			})
		}
		return c.Next()
	}
}
