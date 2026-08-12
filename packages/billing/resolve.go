package billing

import "github.com/zatrano/framework/core"

// From resolves the billing manager from the application container.
func From(app *core.Application) *Manager {
	return core.Resolve[*Manager](app, "billing")
}
