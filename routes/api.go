package routes

import (
	"github.com/zatrano/framework/app/http/controllers/api"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/routing"
)

// API registers application API routes.
func API(app *core.Application) {
	router := app.Router()

	router.Name("api.", func(r *routing.Router) {
		r.Group("/api", func(r *routing.Router) {
			routing.Controller(r, &api.HomeController{}, func(r *routing.Router, c *api.HomeController) {
				r.Get("/", c.Index).As("home")
			})
		})
	})
}
