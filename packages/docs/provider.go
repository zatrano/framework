package docs

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "docs",
		Key:         "docs",
		Description: "Markdown docs repository",
		Factory:     func() kernel.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the docs addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *kernel.Application) error {
	app.Container().Instance("docs", New(app.BasePath("docs")))
	return nil
}

func (p *ServiceProvider) Boot(app *kernel.Application) error { return nil }
