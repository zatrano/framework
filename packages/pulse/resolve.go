package pulse

import "github.com/zatrano/framework/core"

// From resolves the package service from the application container.
func From(app *core.Application) *Dashboard {
	return core.Resolve[*Dashboard](app, "pulse")
}
