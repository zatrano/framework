package bus

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "bus",
		Key:         "bus",
		Description: "Synchronous command bus",
		Factory:     func() kernel.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the bus addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *kernel.Application) error {
	app.Container().Instance("bus", New())
	return nil
}

func (p *ServiceProvider) Boot(app *kernel.Application) error { return nil }
