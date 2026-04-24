package routes

import (
	"github.com/zatrano/framework/app"
	handlers "github.com/zatrano/framework/handlers/auth"
	"github.com/zatrano/framework/middlewares"

	"github.com/gofiber/fiber/v3"
)

func registerAuthRoutes(fiberApp *fiber.App, c *app.Container) {
	authMiddleware := middlewares.AuthMiddleware(c.Auth)

	authHandler := handlers.NewAuthHandler(c.Auth, c.Mail)
	authGroup := fiberApp.Group("/auth")

	authGroup.Get("/login", middlewares.GuestMiddleware, authHandler.ShowLogin)
	authGroup.Post("/login",
		middlewares.GuestMiddleware,
		middlewares.LoginRateLimit(),
		authHandler.Login,
	)

	authGroup.Get("/logout", authMiddleware, authHandler.Logout)
	authGroup.Get("/profile", authMiddleware, authHandler.Profile)
	authGroup.Post("/profile/update-password", authMiddleware, authHandler.UpdatePassword)
	authGroup.Post("/profile/update-info", authMiddleware, authHandler.UpdateInfo)

	authGroup.Get("/register", middlewares.GuestMiddleware, authHandler.ShowRegister)
	authGroup.Post("/register", middlewares.GuestMiddleware, authHandler.Register)

	authGroup.Get("/forgot-password", middlewares.GuestMiddleware, authHandler.ShowForgotPassword)
	authGroup.Post("/forgot-password", middlewares.GuestMiddleware, authHandler.ForgotPassword)

	authGroup.Get("/reset-password", middlewares.GuestMiddleware, authHandler.ShowResetPassword)
	authGroup.Post("/reset-password", middlewares.GuestMiddleware, authHandler.ResetPassword)

	authGroup.Get("/verify-email", middlewares.GuestMiddleware, authHandler.VerifyEmail)
	authGroup.Get("/resend-verification", middlewares.GuestMiddleware, authHandler.ShowResendVerification)
	authGroup.Post("/resend-verification", middlewares.GuestMiddleware, authHandler.ResendVerification)

	authGroup.Get("/oauth/:provider/login", middlewares.GuestMiddleware, authHandler.OAuthLogin)
	authGroup.Get("/oauth/:provider/callback", middlewares.GuestMiddleware, authHandler.OAuthCallback)
}
