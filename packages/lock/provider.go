package lock

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/core"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "lock",
		Key:         "lock",
		Description: "Atomic locks",
		Factory:     func() core.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the lock addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *core.Application) error {
	app.Container().Instance("lock", New())
	return nil
}

func (p *ServiceProvider) Boot(app *core.Application) error { return nil }
