package container

import (
	"fmt"
	"reflect"
	"sync"
)

// Binding represents a service binding in the container.
type Binding struct {
	Concrete any
	Shared   bool
}

// Container is the ZATRANO service container.
type Container struct {
	mu        sync.RWMutex
	bindings  map[string]Binding
	instances map[string]any
	aliases   map[string]string
}

// New creates an empty service container.
func New() *Container {
	return &Container{
		bindings:  make(map[string]Binding),
		instances: make(map[string]any),
		aliases:   make(map[string]string),
	}
}

// Bind registers a non-shared binding.
func (c *Container) Bind(abstract string, concrete any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bindings[abstract] = Binding{Concrete: concrete, Shared: false}
}

// Singleton registers a shared binding.
func (c *Container) Singleton(abstract string, concrete any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bindings[abstract] = Binding{Concrete: concrete, Shared: true}
}

// Instance registers an existing shared instance.
func (c *Container) Instance(abstract string, instance any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.instances[abstract] = instance
	c.bindings[abstract] = Binding{Concrete: instance, Shared: true}
}

// Alias creates an alias for an abstract.
func (c *Container) Alias(abstract, alias string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aliases[alias] = abstract
}

// Bound reports whether an abstract is bound.
func (c *Container) Bound(abstract string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	abstract = c.resolveAlias(abstract)
	_, ok := c.bindings[abstract]
	if ok {
		return true
	}
	_, ok = c.instances[abstract]
	return ok
}

// Make resolves a binding from the container.
func (c *Container) Make(abstract string) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	abstract = c.resolveAlias(abstract)

	if instance, ok := c.instances[abstract]; ok {
		return instance, nil
	}

	binding, ok := c.bindings[abstract]
	if !ok {
		return nil, fmt.Errorf("container: no binding for %q", abstract)
	}

	resolved, err := c.build(binding.Concrete)
	if err != nil {
		return nil, err
	}

	if binding.Shared {
		c.instances[abstract] = resolved
	}

	return resolved, nil
}

// MustMake resolves a binding or panics.
func (c *Container) MustMake(abstract string) any {
	resolved, err := c.Make(abstract)
	if err != nil {
		panic(err)
	}
	return resolved
}

func (c *Container) resolveAlias(abstract string) string {
	if target, ok := c.aliases[abstract]; ok {
		return target
	}
	return abstract
}

func (c *Container) build(concrete any) (any, error) {
	switch v := concrete.(type) {
	case func(*Container) any:
		return v(c), nil
	case func() any:
		return v(), nil
	default:
		rv := reflect.ValueOf(concrete)
		if rv.Kind() == reflect.Func {
			results := rv.Call(nil)
			if len(results) == 0 {
				return nil, fmt.Errorf("container: factory returned no value")
			}
			if len(results) == 2 && !results[1].IsNil() {
				if err, ok := results[1].Interface().(error); ok {
					return nil, err
				}
			}
			return results[0].Interface(), nil
		}
		return concrete, nil
	}
}
