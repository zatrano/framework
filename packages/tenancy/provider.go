package tenancy

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/core"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "tenancy",
		Key:         "tenancy",
		Description: "Multi-tenant resolution",
		Factory:     func() core.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the tenancy addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *core.Application) error {
	mgr := New()
	mgr.Register(Tenant{ID: "acme", Name: "Acme Corp", Domain: "acme.localhost"})
	mgr.Register(Tenant{ID: "globex", Name: "Globex", Domain: "globex.localhost"})
	mgr.SetResolver(mgr.HeaderOrHostResolver())
	mgr.Bootstrapping(func(t *Tenant) error {
		if app.Context() != nil {
			app.Context().Put("tenant.id", t.ID)
			app.Context().Put("tenant.name", t.Name)
		}
		return nil
	})
	app.Container().Instance("tenancy", mgr)
	return nil
}

func (p *ServiceProvider) Boot(app *core.Application) error { return nil }
