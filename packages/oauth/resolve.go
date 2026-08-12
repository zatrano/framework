package oauth

import "github.com/zatrano/framework/core"

// From resolves the OAuth server from the application container.
func From(app *core.Application) *Server {
	return core.Resolve[*Server](app, "oauth")
}
