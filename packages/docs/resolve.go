package docs

import "github.com/zatrano/framework/core"

// From resolves the package service from the application container.
func From(app *core.Application) *Repository {
	return core.Resolve[*Repository](app, "docs")
}
