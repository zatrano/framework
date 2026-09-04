package kernel

import (
	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/kernel/routing"
)

// Resolve loads a typed service from the container.
func Resolve[T any](app contracts.App, key string) T {
	return contracts.Resolve[T](app, key)
}

// ResolveOK loads a typed service or returns the resolution/cast error.
func ResolveOK[T any](app contracts.App, key string) (T, error) {
	return contracts.ResolveOK[T](app, key)
}

// MustResolve loads a typed service or panics.
func MustResolve[T any](app contracts.App, key string) T {
	return contracts.MustResolve[T](app, key)
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
