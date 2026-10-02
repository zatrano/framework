package contracts

import "fmt"

// Resolve loads a typed service from the container.
// Missing bindings return the zero value (typically nil for pointers).
func Resolve[T any](app App, key string) T {
	v, _ := ResolveOK[T](app, key)
	return v
}

// ResolveOK loads a typed service or returns the resolution/cast error.
func ResolveOK[T any](app App, key string) (T, error) {
	var zero T
	if app == nil {
		return zero, fmt.Errorf("container: nil application")
	}
	raw, err := app.Make(key)
	if err != nil {
		return zero, err
	}
	v, ok := raw.(T)
	if !ok {
		return zero, fmt.Errorf("container: %q is %T, not %T", key, raw, zero)
	}
	return v, nil
}

// MustResolve loads a typed service or panics.
func MustResolve[T any](app App, key string) T {
	v, err := ResolveOK[T](app, key)
	if err != nil {
		panic(err)
	}
	return v
}
