package kernel

import (
	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/packages/routing"
)

// Resolve loads a typed service from the container.
func Resolve[T any](app contracts.App, key string) T {
	return contracts.Resolve[T](app, key)
}

type middlewareProvider interface {
	Middleware() routing.MiddlewareFunc
}

func middlewareFrom(app *Application, key string) routing.MiddlewareFunc {
	raw, err := app.Make(key)
	if err != nil {
		return nil
	}
	p, ok := raw.(middlewareProvider)
	if !ok {
		return nil
	}
	return p.Middleware()
}
