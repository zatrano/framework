package security

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/session"
	redisstore "github.com/gofiber/storage/redis/v3"

	"github.com/zatrano/framework/pkg/core"
)

// RegisterSessionAndCSRF wires Redis session store and CSRF for server-rendered forms.
// CSRF is skipped for paths in cfg.Security.CSRFSkipPrefixes and for Bearer JWT requests.
func RegisterSessionAndCSRF(a *core.App, app *fiber.App) {
	cfg := a.Config.Security
	if !cfg.SessionEnabled || a.Redis == nil {
		return
	}

	storage := redisstore.NewFromConnection(a.Redis)
	store := session.NewStore(session.Config{
		Storage:        storage,
		CookieSecure:   cfg.CookieSecure,
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
	})
	a.SessionStore = store

	app.Use(session.New(session.Config{
		Store:          store,
		CookieSecure:   cfg.CookieSecure,
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
	}))

	if !cfg.CSRFEnabled {
		return
	}

	app.Use(csrf.New(csrf.Config{
		Session:        store,
		TrustedOrigins: cfg.TrustedOrigins,
		CookieSecure:   cfg.CookieSecure,
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
		ErrorHandler:   app.Config().ErrorHandler,
		Next:           csrfNext(a),
	}))
}

func csrfNext(a *core.App) func(fiber.Ctx) bool {
	prefixes := a.Config.Security.CSRFSkipPrefixes
	return func(c fiber.Ctx) bool {
		if strings.HasPrefix(strings.TrimSpace(c.Get("Authorization")), "Bearer ") {
			return true
		}
		p := c.Path()
		for _, pre := range prefixes {
			if pre != "" && strings.HasPrefix(p, pre) {
				return true
			}
		}
		return false
	}
}
