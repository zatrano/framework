package routes

import (
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

func init() {
	routing.RegisterWeb(registerHealth)
}

func registerHealth(router *routing.Router) {
	app := currentApp()
	if app == nil || router == nil || app.Health() == nil {
		return
	}

	router.Get("/up", func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"status": "ok"})
	}).As("up")
	router.Get("/health", app.Health().Handler()).As("health")
	router.Get("/api/health", app.Health().Handler()).As("api.health")
}
