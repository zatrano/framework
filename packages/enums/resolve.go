package enums

import "github.com/zatrano/framework/core"

// From resolves the package service from the application container.
func From(app *core.Application) *Registry {
	return core.Resolve[*Registry](app, "enums")
}
