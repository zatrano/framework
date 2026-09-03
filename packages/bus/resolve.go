package bus

import "github.com/zatrano/framework/kernel"

// From resolves the package service from the application container.
func From(app *kernel.Application) *Bus {
	return kernel.Resolve[*Bus](app, "bus")
}
