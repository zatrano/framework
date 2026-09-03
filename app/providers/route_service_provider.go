package providers

import (
	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/routes"
)

// RouteServiceProvider registers application routes.
type RouteServiceProvider struct{}

// Register registers route-related services.
func (p *RouteServiceProvider) Register(app *kernel.Application) error {
	return nil
}

// Boot loads route files.
func (p *RouteServiceProvider) Boot(app *kernel.Application) error {
	routes.Web(app)
	routes.API(app)
	routes.Health(app)
	return nil
}
