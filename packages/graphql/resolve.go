package graphql

import "github.com/zatrano/framework/kernel"

// From resolves the package service from the application container.
func From(app *kernel.Application) *Schema {
	return kernel.Resolve[*Schema](app, "graphql")
}
