package docs

import "github.com/zatrano/framework/kernel"

// From resolves the package service from the application container.
func From(app *kernel.Application) *Repository {
	return kernel.Resolve[*Repository](app, "docs")
}
