package csrfconfig

import (
	"strings"
	"time"

	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/packages/flashmessages"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"go.uber.org/zap"
)

func isProd() bool { return envconfig.IsProd() }

var csrfExemptPaths = []string{"/healthz", "/readyz", "/health"}

func csrfSameSite() string {
	def := "Strict"
	if !isProd() {
		def = "Lax"
	}
	v := strings.ToLower(strings.TrimSpace(envconfig.String("CSRF_COOKIE_SAMESITE", "")))
	switch v {
	case "strict":
		return "Strict"
	case "lax":
		return "Lax"
	case "none":
		return "None"
	default:
		return def
	}
}

func SetupCSRF() fiber.Handler {
	sameSite := csrfSameSite()

	secure := isProd()
	if sameSite == "None" {
		secure = true
	}

	cookieDomain := ""
	if isProd() {
		cookieDomain = envconfig.String("COOKIE_DOMAIN", "")
	}

	// v3: csrf.Config alanları güncellendi
	
	// CookieSameSite artık string (aynı), CookieSessionOnly kaldırıldı
	cfg := csrf.Config{
		KeyLookup:      "header:X-CSRF-Token",
		CookieName:     "csrf_token",
		CookieHTTPOnly: true,
		CookieSecure:   secure,
		CookieSameSite: sameSite,
		CookieDomain:   cookieDomain,
		IdleTimeout:    1 * time.Hour,

		// v3: ErrorHandler imzası: func(fiber.Ctx, error) error
		ErrorHandler: func(c fiber.Ctx, err error) error {
			logconfig.Log.Warn("CSRF validation failed",
				zap.Error(err),
				zap.String("ip", c.IP()),
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
			)
			_ = flashmessages.SetFlashMessage(c, flashmessages.FlashErrorKey,
				"Güvenlik doğrulaması başarısız oldu. Lütfen sayfayı yenileyip tekrar deneyin.")
			return c.Redirect().To("/auth/login") // v3: c.Redirect().To(url)
		},

		// v3: Next imzası: func(fiber.Ctx) bool
		Next: func(c fiber.Ctx) bool {
			// Form'dan gelen CSRF token'ı header'a köprüle
			if c.Get("X-CSRF-Token") == "" {
				if t := c.FormValue("csrf_token"); t != "" {
					c.Request().Header.Set("X-CSRF-Token", t)
				}
			}
			path := c.Path()
			for _, p := range csrfExemptPaths {
				if strings.HasPrefix(path, p) {
					return true
				}
			}
			return false
		},
	}

	logconfig.SLog.Infow("CSRF middleware yapılandırıldı",
		"exempt_paths", csrfExemptPaths,
		"secure", cfg.CookieSecure,
		"samesite", cfg.CookieSameSite,
		"domain", cfg.CookieDomain,
	)
	return csrf.New(cfg)
}
