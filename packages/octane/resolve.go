package octane

import "github.com/zatrano/framework/kernel"

// From resolves the Octane runtime from the application container.
func From(app *kernel.Application) *Runtime {
	return kernel.Resolve[*Runtime](app, "octane")
}
