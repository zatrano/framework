package bus

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/core"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "bus",
		Key:         "bus",
		Description: "Synchronous command bus",
		Factory:     func() core.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the bus addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("bus", New())
	return nil
}

func (p *ServiceProvider) Boot(app *core.Application) error { return nil }
