package features

import "github.com/zatrano/framework/kernel"

// From resolves the feature manager from the application container.
func From(app *kernel.Application) *Manager {
	return kernel.Resolve[*Manager](app, "features")
}
