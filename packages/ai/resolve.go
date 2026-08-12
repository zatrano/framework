package ai

import "github.com/zatrano/framework/core"

// From resolves the AI manager from the application container.
func From(app *core.Application) *Manager {
	return core.Resolve[*Manager](app, "ai")
}
