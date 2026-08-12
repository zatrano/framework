package sitemap

import "github.com/zatrano/framework/core"

// From resolves the package service from the application container.
func From(app *core.Application) *Builder {
	return core.Resolve[*Builder](app, "sitemap")
}
