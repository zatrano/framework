package tenancy

import "github.com/zatrano/framework/kernel"

// From resolves the package service from the application container.
func From(app *kernel.Application) *Manager {
	return kernel.Resolve[*Manager](app, "tenancy")
}
