package routes

import (
	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/packages/http"
)

// Health registers liveness and readiness routes.
func Health(app *kernel.Application) {
	router := app.Router()

	router.Get("/up", func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"status": "ok"})
	}).As("up")
	router.Get("/health", app.Health().Handler()).As("health")
	router.Get("/api/health", app.Health().Handler()).As("api.health")
}
