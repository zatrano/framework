package core

import "github.com/zatrano/framework/packages/routing"

// Resolve loads a typed service from the container.
// Missing bindings return the zero value (typically nil for pointers).
func Resolve[T any](app *Application, key string) T {
	return serviceOf[T](app, key)
}

func serviceOf[T any](app *Application, key string) T {
	var zero T
	if app == nil || app.container == nil {
		return zero
	}
	raw, err := app.container.Make(key)
	if err != nil {
		return zero
	}
	v, ok := raw.(T)
	if !ok {
		return zero
	}
	return v
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
