package orm

import "reflect"

// Fillable models restrict mass assignment to listed columns.
type Fillable interface {
	Fillable() []string
}

// Guarded models block mass assignment for listed columns.
// An empty Guarded list with a Guarded() method that returns []string{"*"} blocks all.
type Guarded interface {
	Guarded() []string
}

func filterMassAssignment[T any](attrs map[string]any) map[string]any {
	if len(attrs) == 0 {
		return attrs
	}
	var zero T
	rv := reflect.ValueOf(&zero).Elem()
	ptr := reflect.New(rv.Type())
	ptr.Elem().Set(rv)

	if fillable, ok := any(ptr.Interface()).(Fillable); ok {
		allowed := map[string]bool{}
		for _, col := range fillable.Fillable() {
			allowed[col] = true
		}
		out := make(map[string]any, len(attrs))
		for k, v := range attrs {
			if allowed[k] {
				out[k] = v
			}
		}
		return out
	}
	if guarded, ok := any(ptr.Interface()).(Guarded); ok {
		blocked := map[string]bool{}
		for _, col := range guarded.Guarded() {
			if col == "*" {
				return map[string]any{}
			}
			blocked[col] = true
		}
		out := make(map[string]any, len(attrs))
		for k, v := range attrs {
			if !blocked[k] {
				out[k] = v
			}
		}
		return out
	}
	return attrs
}
