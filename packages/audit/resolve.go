package audit

import "github.com/zatrano/framework/kernel"

// From resolves the audit manager from the application container.
func From(app *kernel.Application) *Manager {
	return kernel.Resolve[*Manager](app, "audit")
}
