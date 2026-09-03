package geo

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "geo",
		Key:         "geo",
		Description: "Geolocation resolver",
		Factory:     func() kernel.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the geo addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *kernel.Application) error {
	app.Container().Instance("geo", New())
	return nil
}

func (p *ServiceProvider) Boot(app *kernel.Application) error { return nil }
