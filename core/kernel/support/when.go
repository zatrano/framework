package support

// When returns value when condition is true, otherwise the optional fallback (or zero).
func When[T any](condition bool, value T, fallback ...T) T {
	if condition {
		return value
	}
	var zero T
	if len(fallback) > 0 {
		return fallback[0]
	}
	return zero
}

// Unless returns value when condition is false, otherwise the optional fallback (or zero).
func Unless[T any](condition bool, value T, fallback ...T) T {
	return When(!condition, value, fallback...)
}

// Tap calls fn with value and returns value unchanged.
func Tap[T any](value T, fn func(T)) T {
	if fn != nil {
		fn(value)
	}
	return value
}

// Transform maps value through fn.
func Transform[T any, R any](value T, fn func(T) R) R {
	return fn(value)
}
