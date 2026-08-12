package factory

import (
	"fmt"

	"github.com/zatrano/framework/packages/orm"
)

var states = map[string]map[string]Definition{}

// RegisterState registers a named state for a model type.
func RegisterState[T any](name string, definition Definition) {
	mu.Lock()
	defer mu.Unlock()
	key := typeName[T]()
	if states[key] == nil {
		states[key] = map[string]Definition{}
	}
	states[key][name] = definition
}

// HasState reports whether a named state exists for T.
func HasState[T any](name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := states[typeName[T]()][name]
	return ok
}

// ClearStates removes all registered states (mainly for tests).
func ClearStates() {
	mu.Lock()
	defer mu.Unlock()
	states = map[string]map[string]Definition{}
}

func stateOf(typeKey, name string) (Definition, error) {
	mu.RLock()
	defer mu.RUnlock()
	byType, ok := states[typeKey]
	if !ok {
		return nil, fmt.Errorf("factory state [%s] is not defined for [%s]", name, typeKey)
	}
	def, ok := byType[name]
	if !ok {
		return nil, fmt.Errorf("factory state [%s] is not defined for [%s]", name, typeKey)
	}
	return def, nil
}

// Builder fluently builds models with optional states and overrides.
type Builder[T any] struct {
	stateNames []string
	overrides  map[string]any
}

// Of starts a fluent factory builder for T.
func Of[T any]() *Builder[T] {
	return &Builder[T]{}
}

// State applies one or more named states.
func (b *Builder[T]) State(names ...string) *Builder[T] {
	b.stateNames = append(b.stateNames, names...)
	return b
}

// Merge merges attribute overrides.
func (b *Builder[T]) Merge(attrs map[string]any) *Builder[T] {
	if b.overrides == nil {
		b.overrides = map[string]any{}
	}
	for key, value := range attrs {
		b.overrides[key] = value
	}
	return b
}

// Make builds attributes without persisting.
func (b *Builder[T]) Make() (map[string]any, error) {
	attrs, err := Make[T]()
	if err != nil {
		return nil, err
	}
	key := typeName[T]()
	for _, name := range b.stateNames {
		def, err := stateOf(key, name)
		if err != nil {
			return nil, err
		}
		for k, v := range def() {
			attrs[k] = v
		}
	}
	for k, v := range b.overrides {
		attrs[k] = v
	}
	return attrs, nil
}

// MakeMany builds many attribute maps.
func (b *Builder[T]) MakeMany(count int) ([]map[string]any, error) {
	out := make([]map[string]any, 0, count)
	for i := 0; i < count; i++ {
		attrs, err := b.Make()
		if err != nil {
			return nil, err
		}
		out = append(out, attrs)
	}
	return out, nil
}

// Create persists a model.
func (b *Builder[T]) Create() (*T, error) {
	attrs, err := b.Make()
	if err != nil {
		return nil, err
	}
	return orm.Create[T](attrs)
}

// CreateMany persists many models.
func (b *Builder[T]) CreateMany(count int) ([]T, error) {
	out := make([]T, 0, count)
	for i := 0; i < count; i++ {
		model, err := b.Create()
		if err != nil {
			return nil, err
		}
		out = append(out, *model)
	}
	return out, nil
}
