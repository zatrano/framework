package bus

import "github.com/zatrano/framework/core"

// From resolves the package service from the application container.
func From(app *core.Application) *Bus {
	return core.Resolve[*Bus](app, "bus")
}
