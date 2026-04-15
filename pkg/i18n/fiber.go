package i18n

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/framework/pkg/config"
)

// Locals keys for Fiber request context.
const (
	LocalsBundle = "zatrano.i18n.bundle"
	LocalsLocale = "zatrano.locale"
)

// Middleware resolves the active locale and stores Bundle + tag on c.Locals (use with Bundle from bootstrap).
func Middleware(b *Bundle, cfg config.I18n) fiber.Handler {
	cookieName := strings.TrimSpace(cfg.CookieName)
	if cookieName == "" {
		cookieName = "zatrano_lang"
	}
	qk := strings.TrimSpace(cfg.QueryKey)
	if qk == "" {
		qk = "lang"
	}
	return func(c fiber.Ctx) error {
		c.Locals(LocalsBundle, b)
		loc := b.PickLocale(
			c.Get("Accept-Language"),
			c.Query(qk),
			c.Cookies(cookieName),
		)
		c.Locals(LocalsLocale, loc)
		return c.Next()
	}
}

// Locale returns the resolved locale tag for this request, or "" if i18n middleware did not run.
func Locale(c fiber.Ctx) string {
	s, _ := c.Locals(LocalsLocale).(string)
	return s
}

// T translates key using the bundle and locale on context; falls back to key if bundle missing.
func T(c fiber.Ctx, key string) string {
	b, _ := c.Locals(LocalsBundle).(*Bundle)
	if b == nil {
		return key
	}
	loc, _ := c.Locals(LocalsLocale).(string)
	if loc == "" {
		loc = b.DefaultLocale()
	}
	return b.T(loc, key)
}

// Tf translates key then renders it as a text/template with data (see Bundle.Format).
// If i18n middleware is not active, returns key and a nil error (same as T).
func Tf(c fiber.Ctx, key string, data any) (string, error) {
	b, _ := c.Locals(LocalsBundle).(*Bundle)
	if b == nil {
		return key, nil
	}
	loc, _ := c.Locals(LocalsLocale).(string)
	if loc == "" {
		loc = b.DefaultLocale()
	}
	return b.Format(loc, key, data)
}
