package routes

import (
	"github.com/zatrano/framework/app/http/controllers/web"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/api"
	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/authorization"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// RegisterDashboardWeb registers the management dashboard under /dashboard.
// Requires make:auth (User + auth middleware) and DashboardServiceProvider gates.
func RegisterDashboardWeb(app *core.Application) {
	router := app.Router()
	dash := &web.DashboardController{App: app}
	users := &web.DashboardUsersController{App: app}
	notifs := &web.DashboardNotificationsController{App: app}
	roles := &web.DashboardRolesController{App: app}
	rbac := &web.DashboardRBACController{App: app}
	settings := &web.DashboardSettingsController{App: app}
	analytics := &web.DashboardAnalyticsController{App: app}

	gate := authorization.From(app)
	current := func(req *http.Request) any { return auth.From(app).User(req) }
	protect := []routing.MiddlewareFunc{
		auth.Middleware(auth.From(app)),
		authorization.Middleware(gate, current, "dashboard.access"),
	}

	router.Group("/dashboard", func(r *routing.Router) {
		r.Get("/", dash.Home).As("dashboard.home")

		r.Get("/users", users.Index).As("dashboard.users")
		r.Get("/users/create", users.CreateForm).As("dashboard.users.create")
		r.Post("/users", users.Store).As("dashboard.users.store")
		r.Get("/users/{id}/edit", users.EditForm).As("dashboard.users.edit").WhereNumber("id")
		r.Post("/users/{id}", users.Update).As("dashboard.users.update").WhereNumber("id")
		r.Post("/users/{id}/delete", users.Delete).As("dashboard.users.delete").WhereNumber("id")
		r.Post("/users/{id}/impersonate", users.Impersonate).As("dashboard.users.impersonate").WhereNumber("id")
		r.Post("/impersonation/stop", users.StopImpersonation).As("dashboard.impersonation.stop")

		r.Get("/notifications", notifs.Index).As("dashboard.notifications")
		r.Get("/notifications/send", notifs.SendForm).As("dashboard.notifications.send")
		r.Post("/notifications/send", notifs.Send).As("dashboard.notifications.send.store")
		r.Get("/notifications/bulk", notifs.BulkForm).As("dashboard.notifications.bulk")
		r.Post("/notifications/bulk", notifs.Bulk).As("dashboard.notifications.bulk.store")
		r.Post("/notifications/read-all", notifs.MarkAllRead).As("dashboard.notifications.read_all")
		r.Post("/notifications/{id}/read", notifs.MarkRead).As("dashboard.notifications.read").WhereNumber("id")

		r.Get("/roles", roles.Index).As("dashboard.roles")
		r.Get("/roles/create", roles.CreateForm).As("dashboard.roles.create")
		r.Post("/roles", roles.Store).As("dashboard.roles.store")
		r.Get("/roles/{id}/edit", roles.EditForm).As("dashboard.roles.edit").WhereNumber("id")
		r.Post("/roles/{id}", roles.Update).As("dashboard.roles.update").WhereNumber("id")
		r.Post("/roles/{id}/delete", roles.Delete).As("dashboard.roles.delete").WhereNumber("id")

		r.Get("/rbac", rbac.Matrix).As("dashboard.rbac")
		r.Post("/rbac", rbac.Save).As("dashboard.rbac.save")

		r.Get("/settings", settings.Index).As("dashboard.settings")
		r.Post("/settings", settings.Save).As("dashboard.settings.save")

		r.Get("/analytics", analytics.Index).As("dashboard.analytics")
		r.Get("/analytics/overview", analytics.OverviewAPI).As("dashboard.analytics.overview")
	}, protect...)
}

// RegisterDashboardAPI registers versioned JSON routes under /api/v1.
func RegisterDashboardAPI(app *core.Application) {
	apiCtrl := &web.DashboardAPIController{App: app}
	analytics := &web.DashboardAnalyticsController{App: app}

	gate := authorization.From(app)
	current := func(req *http.Request) any { return auth.From(app).User(req) }
	protect := []routing.MiddlewareFunc{
		auth.Middleware(auth.From(app)),
		authorization.Middleware(gate, current, "dashboard.access"),
	}

	api.Version(app.Router(), "v1", func(r *routing.Router) {
		r.Get("/users", apiCtrl.UsersIndex).As("api.v1.users")
		r.Get("/users/{id}", apiCtrl.UsersShow).As("api.v1.users.show").WhereNumber("id")
		r.Get("/roles", apiCtrl.RolesIndex).As("api.v1.roles")
		r.Get("/settings", apiCtrl.SettingsIndex).As("api.v1.settings")
		r.Get("/notifications", apiCtrl.NotificationsIndex).As("api.v1.notifications")
		r.Get("/analytics/overview", analytics.OverviewAPI).As("api.v1.analytics.overview")
	}, protect...)
}
