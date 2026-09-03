package providers

import (
	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/packages/routing"
	"github.com/zatrano/framework/routes"
)

// RouteServiceProvider registers application routes.
type RouteServiceProvider struct{}

// Register registers route-related services.
func (p *RouteServiceProvider) Register(app *kernel.Application) error {
	return nil
}

// Boot applies self-registered web and API route groups.
func (p *RouteServiceProvider) Boot(app *kernel.Application) error {
	routes.Use(app)
	router, _ := app.Router().(*routing.Router)
	routing.ApplyWeb(router)
	routing.ApplyAPI(router)
	return nil
}
