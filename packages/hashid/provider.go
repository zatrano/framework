package hashid

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/packages/env"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "hashid",
		Key:         "hashid",
		Description: "Obfuscated public IDs",
		Factory:     func() kernel.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the hashid addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *kernel.Application) error {
	app.Container().Instance("hashid", New(
		env.Get("HASHID_SALT", app.Config().GetString("app.key", "zatrano")),
		env.GetInt("HASHID_MIN_LENGTH", 8),
	))
	return nil
}

func (p *ServiceProvider) Boot(app *kernel.Application) error { return nil }
