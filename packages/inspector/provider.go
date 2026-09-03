package inspector

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/core"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "inspector",
		Key:         "inspector",
		Description: "Request inspector toolbar data",
		Factory:     func() core.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the inspector addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("inspector", New(200))
	return nil
}

func (p *ServiceProvider) Boot(app *core.Application) error { return nil }
