package web

import (
	"github.com/zatrano/framework/app/models"
	"github.com/zatrano/framework/core"
	. "github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/notification"
	"github.com/zatrano/framework/packages/orm"
)

// DashboardAPIController exposes versioned JSON endpoints under /api/v1.
type DashboardAPIController struct {
	App *core.Application
}

func (c *DashboardAPIController) UsersIndex(req *Request) *Response {
	users, err := orm.Query[models.User]().OrderByDesc("id").Limit(100).Get()
	if err != nil {
		return JSON(map[string]any{"message": err.Error()}).Status(500)
	}
	return JSON(map[string]any{"data": users})
}

func (c *DashboardAPIController) UsersShow(req *Request) *Response {
	id, err := dashboardParseID(req, "id")
	if err != nil {
		return JSON(map[string]any{"message": "not found"}).Status(404)
	}
	user, err := orm.Find[models.User](id)
	if err != nil || user == nil {
		return JSON(map[string]any{"message": "not found"}).Status(404)
	}
	return JSON(map[string]any{"data": user})
}


func (c *DashboardAPIController) RolesIndex(req *Request) *Response {
	roles, err := orm.Query[models.Role]().OrderBy("id").Get()
	if err != nil {
		return JSON(map[string]any{"message": err.Error()}).Status(500)
	}
	return JSON(map[string]any{"data": roles})
}


func (c *DashboardAPIController) SettingsIndex(req *Request) *Response {
	settings, err := orm.Query[models.Setting]().OrderBy("key").Get()
	if err != nil {
		return JSON(map[string]any{"message": err.Error()}).Status(500)
	}
	return JSON(map[string]any{"data": settings})
}


func (c *DashboardAPIController) NotificationsIndex(req *Request) *Response {
	u := dashboardCurrentUser(req, c.App)
	if u == nil {
		return JSON(map[string]any{"message": "Unauthenticated."}).Status(401)
	}
	store := notification.From(c.App).Store()
	if store == nil {
		return JSON(map[string]any{"data": []any{}})
	}
	items, err := store.ListFor(dashboardUserIDString(u), 100)
	if err != nil {
		return JSON(map[string]any{"message": err.Error()}).Status(500)
	}
	return JSON(map[string]any{"data": items})
}

