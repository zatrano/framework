package routes

import (
	"github.com/zatrano/framework/app/http/controllers/api"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/core/routing"
)

// API registers application API routes.
func API(app *core.Application) {
	router := app.Router()

	router.Name("api.", func(r *routing.Router) {
		r.Group("/api", func(r *routing.Router) {
			routing.Controller(r, &api.HomeController{}, func(r *routing.Router, c *api.HomeController) {
				r.Get("/", c.Index).As("home")
			})

			notifications := &api.NotificationsController{App: app}
			routing.Controller(r, notifications, func(r *routing.Router, c *api.NotificationsController) {
				r.Get("/notifications", c.Index).As("notifications.index")
				r.Post("/notifications/send", c.Send).As("notifications.send")
				r.Post("/notifications/bulk", c.Bulk).As("notifications.bulk")
				r.Post("/notifications/read-all", c.MarkAllRead).As("notifications.read_all")
				r.Post("/notifications/{id}/read", c.MarkRead).As("notifications.read")
			})
		})
	})
}
