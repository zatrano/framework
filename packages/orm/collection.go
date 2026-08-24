package orm

// Collection is a thin helper over []T for relation loading / mapping DX.
type Collection[T any] []T

// Collect wraps a slice as a Collection.
func Collect[T any](items []T) Collection[T] { return Collection[T](items) }

// Load runs a loader against the collection (mutates elements in place).
func (c Collection[T]) Load(loader func([]T) error) error {
	if loader == nil || len(c) == 0 {
		return nil
	}
	items := []T(c)
	if err := loader(items); err != nil {
		return err
	}
	copy(c, items)
	return nil
}

// Pluck extracts a column via attribute name into []any.
func (c Collection[T]) Pluck(column string) ([]any, error) {
	out := make([]any, 0, len(c))
	for i := range c {
		v, err := attribute(&c[i], column)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// Map transforms each model.
func (c Collection[T]) Map(fn func(T) T) Collection[T] {
	if fn == nil {
		return c
	}
	out := make(Collection[T], len(c))
	for i := range c {
		out[i] = fn(c[i])
	}
	return out
}

// Each invokes fn for each element.
func (c Collection[T]) Each(fn func(*T) error) error {
	if fn == nil {
		return nil
	}
	for i := range c {
		if err := fn(&c[i]); err != nil {
			return err
		}
	}
	return nil
}

// Slice returns the underlying slice.
func (c Collection[T]) Slice() []T { return []T(c) }
