package web

import (
	"github.com/zatrano/framework/app/models"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/flash"
	. "github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/orm"
	"github.com/zatrano/framework/packages/validation"
)

const pathDashboardRoles = "/dashboard/roles"

// DashboardRolesController manages roles.
type DashboardRolesController struct {
	App *core.Application
}

func (c *DashboardRolesController) Index(req *Request) *Response {
	roles, err := orm.Query[models.Role]().OrderBy("id").Get()
	if err != nil {
		return flash.WithError(req, dashboardLang(c.App, "dashboard.load_failed"), "/dashboard")
	}
	return View("dashboard/roles/index", map[string]any{
		"title": dashboardLang(c.App, "dashboard.nav_roles"),
		"roles": roles,
		"user":  dashboardCurrentUser(req, c.App),
	})
}

func (c *DashboardRolesController) CreateForm(req *Request) *Response {
	return View("dashboard/roles/create", map[string]any{
		"title": dashboardLang(c.App, "dashboard.new_role"),
		"user":  dashboardCurrentUser(req, c.App),
		"name":  flash.OldValue(req, "name"),
		"slug":  flash.OldValue(req, "slug"),
	})
}

func (c *DashboardRolesController) Store(req *Request) *Response {
	v := validation.Make(req.All(), map[string]string{
		"name": "required",
		"slug": "required",
	})
	if v.Fails() {
		flash.Old(req, req.Only("name", "slug"))
		return flash.WithError(req, dashboardLang(c.App, "dashboard.load_failed"), pathDashboardRoles+"/create")
	}
	role := &models.Role{Name: req.Input("name"), Slug: req.Input("slug")}
	if err := orm.Save(role); err != nil {
		return flash.WithError(req, err.Error(), pathDashboardRoles+"/create")
	}
	return flash.WithSuccess(req, dashboardLang(c.App, "dashboard.created"), pathDashboardRoles)
}

func (c *DashboardRolesController) EditForm(req *Request) *Response {
	id, err := dashboardParseID(req, "id")
	if err != nil {
		return flash.WithError(req, dashboardLang(c.App, "dashboard.not_found"), pathDashboardRoles)
	}
	item, err := orm.Find[models.Role](id)
	if err != nil || item == nil {
		return flash.WithError(req, dashboardLang(c.App, "dashboard.not_found"), pathDashboardRoles)
	}
	return View("dashboard/roles/edit", map[string]any{
		"title": dashboardLang(c.App, "dashboard.edit"),
		"item":  item,
		"user":  dashboardCurrentUser(req, c.App),
	})
}

func (c *DashboardRolesController) Update(req *Request) *Response {
	id, err := dashboardParseID(req, "id")
	if err != nil {
		return flash.WithError(req, dashboardLang(c.App, "dashboard.not_found"), pathDashboardRoles)
	}
	item, err := orm.Find[models.Role](id)
	if err != nil || item == nil {
		return flash.WithError(req, dashboardLang(c.App, "dashboard.not_found"), pathDashboardRoles)
	}
	item.Name = req.Input("name")
	item.Slug = req.Input("slug")
	if err := orm.Save(item); err != nil {
		return flash.WithError(req, err.Error(), pathDashboardRoles+"/"+req.Route("id")+"/edit")
	}
	return flash.WithSuccess(req, dashboardLang(c.App, "dashboard.saved"), pathDashboardRoles)
}

func (c *DashboardRolesController) Delete(req *Request) *Response {
	id, err := dashboardParseID(req, "id")
	if err != nil {
		return flash.WithError(req, dashboardLang(c.App, "dashboard.not_found"), pathDashboardRoles)
	}
	item, err := orm.Find[models.Role](id)
	if err != nil || item == nil {
		return flash.WithError(req, dashboardLang(c.App, "dashboard.not_found"), pathDashboardRoles)
	}
	if _, err := orm.DeleteModel(item); err != nil {
		return flash.WithError(req, err.Error(), pathDashboardRoles)
	}
	return flash.WithSuccess(req, dashboardLang(c.App, "dashboard.deleted"), pathDashboardRoles)
}
