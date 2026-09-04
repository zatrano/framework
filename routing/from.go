package routing

import "github.com/zatrano/framework/contracts"

// From resolves the concrete router from the application container.
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
