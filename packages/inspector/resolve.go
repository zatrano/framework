package inspector

import "github.com/zatrano/framework/core"

// From resolves the package service from the application container.
func From(app *core.Application) *Manager {
	return core.Resolve[*Manager](app, "inspector")
}
