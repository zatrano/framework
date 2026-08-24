package orm

import (
	"fmt"
	"reflect"
)

// AttributeGetter provides computed/read accessors for ToMap and Attribute().
type AttributeGetter interface {
	GetAttribute(name string) (any, bool)
}

// AttributeSetter provides write mutators applied during Create/Save mass assignment.
type AttributeSetter interface {
	SetAttribute(name string, value any) (any, bool)
}

// Appends lists virtual attribute names merged into ToMap via AttributeGetter.
type Appends interface {
	Appends() []string
}

// Attribute returns a model attribute, preferring AttributeGetter accessors.
func Attribute[T any](model *T, name string) (any, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}
	if getter, ok := any(model).(AttributeGetter); ok {
		if v, found := getter.GetAttribute(name); found {
			return v, nil
		}
		if v, found := getter.GetAttribute(toSnake(name)); found {
			return v, nil
		}
	}
	return attribute(model, name)
}

// applyMutators runs AttributeSetter for each attr before persistence.
func applyMutators[T any](attrs map[string]any) map[string]any {
	if len(attrs) == 0 {
		return attrs
	}
	var zero T
	ptr := reflect.New(reflect.TypeOf(zero)).Interface()
	setter, ok := ptr.(AttributeSetter)
	if !ok {
		return attrs
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		if nv, handled := setter.SetAttribute(k, v); handled {
			out[k] = nv
			continue
		}
		if nv, handled := setter.SetAttribute(toSnake(k), v); handled {
			out[k] = nv
			continue
		}
		out[k] = v
	}
	return out
}

// applyAccessors merges AttributeGetter / Appends values into a serialization map.
func applyAccessors[T any](model *T, raw map[string]any) map[string]any {
	if model == nil || raw == nil {
		return raw
	}
	getter, ok := any(model).(AttributeGetter)
	if !ok {
		return raw
	}
	out := make(map[string]any, len(raw)+4)
	for k, v := range raw {
		if av, found := getter.GetAttribute(k); found {
			out[k] = av
			continue
		}
		out[k] = v
	}
	if appends, ok := any(model).(Appends); ok {
		for _, name := range appends.Appends() {
			if av, found := getter.GetAttribute(name); found {
				out[name] = av
			}
		}
	}
	return out
}
