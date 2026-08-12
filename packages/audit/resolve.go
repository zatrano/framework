package audit

import "github.com/zatrano/framework/core"

// From resolves the audit manager from the application container.
func From(app *core.Application) *Manager {
	return core.Resolve[*Manager](app, "audit")
}
