package sitemap

import "github.com/zatrano/framework/kernel"

// From resolves the package service from the application container.
func From(app *kernel.Application) *Builder {
	return kernel.Resolve[*Builder](app, "sitemap")
}
