// Package routing is the typed HTTP router.
//
// contracts.Router is the dependency-neutral ABI (handlers are untyped so
// contracts does not import this package). Application code should use From(app)
// for HandlerFunc and MiddlewareFunc.
package routing

import "github.com/zatrano/framework/v2/contracts"

// From returns the typed router. Prefer this over App.Router() when registering
// HandlerFunc or MiddlewareFunc values.
func From(app contracts.App) *Router {
	if app == nil {
		return nil
	}
	raw, err := app.Make("router")
	if err != nil {
		return nil
	}
	r, _ := raw.(*Router)
	return r
}
