package graphql

import "github.com/zatrano/framework/core"

// From resolves the package service from the application container.
func From(app *core.Application) *Schema {
	return core.Resolve[*Schema](app, "graphql")
}
