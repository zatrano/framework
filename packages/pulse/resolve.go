package pulse

import "github.com/zatrano/framework/kernel"

// From resolves the package service from the application container.
func From(app *kernel.Application) *Dashboard {
	return kernel.Resolve[*Dashboard](app, "pulse")
}
