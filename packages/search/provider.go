package search

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "search",
		Key:         "search",
		Description: "In-memory search engine",
		Factory:     func() kernel.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the search addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *kernel.Application) error {
	app.Container().Instance("search", New(NewMemoryEngine()))
	return nil
}

func (p *ServiceProvider) Boot(app *kernel.Application) error { return nil }
