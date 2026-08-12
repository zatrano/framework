package octane

import "github.com/zatrano/framework/core"

// From resolves the Octane runtime from the application container.
func From(app *core.Application) *Runtime {
	return core.Resolve[*Runtime](app, "octane")
}
