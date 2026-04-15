package server

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/zatrano/framework/pkg/core"
	"github.com/zatrano/framework/pkg/security"
)

func registerAPI(a *core.App, app *fiber.App) {
	api := app.Group("/api/v1")

	api.Get("/public/ping", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "pong",
		})
	})

	secret := strings.TrimSpace(a.Config.Security.JWTSecret)
	if secret == "" {
		return
	}

	if a.Config.Security.DemoTokenEndpoint {
		api.Post("/auth/token", demoTokenHandler(a))
	}

	priv := api.Group("/private", security.JWTMiddleware(a.Config))
	priv.Get("/me", func(c fiber.Ctx) error {
		claims, _ := c.Locals(security.ClaimsKey()).(jwtlib.MapClaims)
		return c.JSON(fiber.Map{"claims": claims})
	})
}

func demoTokenHandler(a *core.App) fiber.Handler {
	return func(c fiber.Ctx) error {
		var body struct {
			Sub string `json:"sub"`
		}
		if err := c.Bind().Body(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json body")
		}
		sub := strings.TrimSpace(body.Sub)
		if sub == "" {
			return fiber.NewError(fiber.StatusBadRequest, "sub is required")
		}
		tok, err := security.SignAccessToken(a.Config, sub, nil)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{
			"access_token": tok,
			"token_type":   "Bearer",
			"expires_in":   int(a.Config.Security.JWTExpiry.Seconds()),
		})
	}
}
