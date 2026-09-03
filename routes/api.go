package routes

import (
	"github.com/zatrano/framework/app/http/controllers/api"
	"github.com/zatrano/framework/packages/routing"
)

func init() {
	routing.RegisterAPI(registerAPI)
}

func registerAPI(router *routing.Router) {
	if router == nil {
		return
	}

	router.Name("api.", func(r *routing.Router) {
		r.Group("/api", func(r *routing.Router) {
			routing.Controller(r, &api.HomeController{}, func(r routing.RouteRegistrar, c *api.HomeController) {
				r.Get("/", c.Index).As("home")
			})
		})
	})
}
