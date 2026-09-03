package octane

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/packages/env"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "octane",
		Key:         "octane",
		Description: "Concurrent runtime metrics",
		Factory:     func() kernel.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the octane addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *kernel.Application) error {
	app.Container().Instance("octane", New(env.GetInt("OCTANE_WORKERS", 0)))
	return nil
}

func (p *ServiceProvider) Boot(app *kernel.Application) error { return nil }
