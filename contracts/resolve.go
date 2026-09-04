package contracts

// Resolve loads a typed service from the container.
// Missing bindings return the zero value (typically nil for pointers).
func Resolve[T any](app App, key string) T {
	var zero T
	if app == nil {
		return zero
	}
	raw, err := app.Make(key)
	if err != nil {
		return zero
	}
	v, ok := raw.(T)
	if !ok {
		return zero
	}
	return v
}
