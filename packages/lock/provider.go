package lock

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "lock",
		Key:         "lock",
		Description: "Atomic locks",
		Factory:     func() kernel.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the lock addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *kernel.Application) error {
	app.Container().Instance("lock", New())
	return nil
}

func (p *ServiceProvider) Boot(app *kernel.Application) error { return nil }
