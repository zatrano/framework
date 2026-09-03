package geo

import "github.com/zatrano/framework/kernel"

// From resolves the package service from the application container.
func From(app *kernel.Application) *Resolver {
	return kernel.Resolve[*Resolver](app, "geo")
}
