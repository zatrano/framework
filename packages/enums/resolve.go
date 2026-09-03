package enums

import "github.com/zatrano/framework/kernel"

// From resolves the package service from the application container.
func From(app *kernel.Application) *Registry {
	return kernel.Resolve[*Registry](app, "enums")
}
