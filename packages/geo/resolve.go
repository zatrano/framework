package geo

import "github.com/zatrano/framework/core"

// From resolves the package service from the application container.
func From(app *core.Application) *Resolver {
	return core.Resolve[*Resolver](app, "geo")
}
