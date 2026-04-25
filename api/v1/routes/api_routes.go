// Package routes — REST API v1 rota tanımları.
// Tüm uç noktalar /api/v1 altında, JWT ile korunur.
// CORS tüm API grubuna uygulanır.
package routes

import (
	"github.com/zatrano/framework/api/v1/handlers"
	"github.com/zatrano/framework/app"
	"github.com/zatrano/framework/middlewares"

	"github.com/gofiber/fiber/v3"
)

// main.go fiber.Config.BodyLimit ile aynı üst sınır (JSON API gövdeleri).
const apiMaxBodyBytes = 10 * 1024 * 1024

func SetupAPIRoutes(fiberApp *fiber.App, c *app.Container) {
	// Sıra: CORS → hız sınırı → JSON Content-Type (POST/PUT/PATCH) → gövde boyutu → iş mantığı
	api := fiberApp.Group("/api/v1",
		middlewares.CORS(),
		middlewares.APIRateLimit(),
		middlewares.ContentTypeEnforcer("application/json"),
		middlewares.RequestSizeLimiter(apiMaxBodyBytes),
	)

	// ── Auth (herkese açık) ────────────────────────────────────────────────────────
	authHandler := handlers.NewAuthAPIHandler(c.Auth, c.JWT)
	auth := api.Group("/auth")
	auth.Post("/login", authHandler.Login)
	auth.Post("/register", authHandler.Register)
	auth.Post("/refresh", authHandler.Refresh)
	auth.Post("/verify-email", authHandler.VerifyEmail)
	auth.Post("/resend-verification", authHandler.ResendVerification)
	auth.Post("/forgot-password", authHandler.ForgotPassword)
	auth.Post("/reset-password", authHandler.ResetPassword)

	// ── Auth (korumalı) ─────────────────────────────────────────────────────
	authProtected := auth.Group("", middlewares.JWTAuth())
	authProtected.Get("/me", authHandler.Me)
	authProtected.Post("/logout", authHandler.Logout)

	// ── Kullanıcı (type=2: standart kullanıcılar) ───────────────────────────────────────
	userHandler := handlers.NewUserAPIHandler(c.Auth)
	userGroup := api.Group("/user",
		middlewares.JWTAuth(),
		middlewares.JWTTypeMiddleware(1, 2), // hem admin hem user
	)
	userGroup.Get("/profile", userHandler.Profile)
	userGroup.Put("/profile", userHandler.UpdateProfile)
	userGroup.Put("/password", userHandler.ChangePassword)

	// ── Yönetici (type=1: yalnızca admin) ────────────────────────────────────────────
	adminUserHandler := handlers.NewAdminUserAPIHandler(c.User)
	adminGroup := api.Group("/admin",
		middlewares.JWTAuth(),
		middlewares.JWTTypeMiddleware(1),
	)
	// Kullanıcı yönetimi
	adminGroup.Get("/users", adminUserHandler.ListUsers)
	adminGroup.Get("/users/:id", adminUserHandler.GetUser)
	adminGroup.Post("/users", adminUserHandler.CreateUser)
	adminGroup.Put("/users/:id", adminUserHandler.UpdateUser)
	adminGroup.Delete("/users/:id", adminUserHandler.DeleteUser)
}
