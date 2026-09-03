package oauth

import "github.com/zatrano/framework/kernel"

// From resolves the OAuth server from the application container.
func From(app *kernel.Application) *Server {
	return kernel.Resolve[*Server](app, "oauth")
}
