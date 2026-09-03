package geo

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/core"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "geo",
		Key:         "geo",
		Description: "Geolocation resolver",
		Factory:     func() core.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the geo addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("geo", New())
	return nil
}

func (p *ServiceProvider) Boot(app *core.Application) error { return nil }
