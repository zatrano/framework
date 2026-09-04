package ai

import "github.com/zatrano/framework/contracts"

// From resolves the AI manager from the application container.
func From(app contracts.App) *Manager {
	return contracts.Resolve[*Manager](app, "ai")
}
