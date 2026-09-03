package features

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/core"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "features",
		Key:         "features",
		Description: "Feature flags and gradual rollouts",
		Factory:     func() core.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the features addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *core.Application) error {
	mgr := New()
	mgr.Activate("welcome_banner")
	mgr.Deactivate("beta_editor")
	mgr.Rollout("new_dashboard", 25)
	app.Container().Instance("features", mgr)
	return nil
}

func (p *ServiceProvider) Boot(app *core.Application) error { return nil }
