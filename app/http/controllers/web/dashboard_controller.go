package web

import (
	"github.com/zatrano/framework/app/models"
	"github.com/zatrano/framework/core"
	. "github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/notification"
	"github.com/zatrano/framework/packages/orm"
)

// DashboardController serves the dashboard home.
type DashboardController struct {
	App *core.Application
}

// Home shows summary cards for enabled modules.
func (c *DashboardController) Home(req *Request) *Response {
	usersCount := int64(0)
	rolesCount := int64(0)
	unreadCount := 0
	if n, err := orm.Query[models.User]().Count(); err == nil {
		usersCount = n
	}
	if n, err := orm.Query[models.Role]().Count(); err == nil {
		rolesCount = n
	}
	if u := dashboardCurrentUser(req, c.App); u != nil {
		if store := notification.From(c.App).Store(); store != nil {
			if items, err := store.UnreadFor(dashboardUserIDString(u), 200); err == nil {
				unreadCount = len(items)
			}
		}
	}

	return View("dashboard/home", map[string]any{
		"title":        dashboardLang(c.App, "dashboard.nav_home"),
		"user":         dashboardCurrentUser(req, c.App),
		"users_count":  usersCount,
		"roles_count":  rolesCount,
		"unread_count": unreadCount,
	})
}
