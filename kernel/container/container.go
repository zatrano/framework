// Package container is a factory-based service container.
//
// It does not perform constructor-argument reflection or automatic
// dependency graphs. Bind a factory (or an instance) and resolve it with Make.
// Nested Make calls from inside a factory are supported and must not deadlock.
package container

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// Binding represents a service binding in the container.
type Binding struct {
	Concrete any
	Shared   bool
}

type sharedSlot struct {
	ready chan struct{}
	val   any
	err   error
}

// Container is the ZATRANO factory-based service container.
type Container struct {
	mu        sync.Mutex
	bindings  map[string]Binding
	instances map[string]any
	aliases   map[string]string
	slots     map[string]*sharedSlot
	chains    sync.Map // goroutine id -> []string
}

// New creates an empty service container.
func New() *Container {
	return &Container{
		bindings:  make(map[string]Binding),
		instances: make(map[string]any),
		aliases:   make(map[string]string),
		slots:     make(map[string]*sharedSlot),
	}
}

// Bind registers a non-shared binding.
func (c *Container) Bind(abstract string, concrete any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.instances, abstract)
	delete(c.slots, abstract)
	c.bindings[abstract] = Binding{Concrete: concrete, Shared: false}
}

// Singleton registers a shared binding.
func (c *Container) Singleton(abstract string, concrete any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.instances, abstract)
	delete(c.slots, abstract)
	c.bindings[abstract] = Binding{Concrete: concrete, Shared: true}
}

// Instance registers an existing shared instance.
func (c *Container) Instance(abstract string, instance any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.slots, abstract)
	c.instances[abstract] = instance
	c.bindings[abstract] = Binding{Concrete: instance, Shared: true}
}

// Alias creates an alias that resolves to abstract.
func (c *Container) Alias(abstract, alias string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aliases[alias] = abstract
}

// Bound reports whether an abstract is bound.
func (c *Container) Bound(abstract string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	resolved, err := c.resolveAlias(abstract)
	if err != nil {
		return false
	}
	if _, ok := c.bindings[resolved]; ok {
		return true
	}
	_, ok := c.instances[resolved]
	return ok
}

// Make resolves a binding from the container.
func (c *Container) Make(abstract string) (any, error) {
	c.mu.Lock()
	resolved, err := c.resolveAlias(abstract)
	if err != nil {
		c.mu.Unlock()
		return nil, err
	}
	abstract = resolved

	if instance, ok := c.instances[abstract]; ok {
		c.mu.Unlock()
		return instance, nil
	}

	binding, ok := c.bindings[abstract]
	if !ok {
		c.mu.Unlock()
		return nil, fmt.Errorf("container: no binding for %q", abstract)
	}

	if err := c.pushResolving(abstract); err != nil {
		c.mu.Unlock()
		return nil, err
	}

	if !binding.Shared {
		concrete := binding.Concrete
		c.mu.Unlock()
		resolvedVal, buildErr := c.build(concrete)
		c.popResolving()
		return resolvedVal, buildErr
	}

	if slot, ok := c.slots[abstract]; ok {
		c.mu.Unlock()
		<-slot.ready
		c.popResolving()
		return slot.val, slot.err
	}

	slot := &sharedSlot{ready: make(chan struct{})}
	c.slots[abstract] = slot
	concrete := binding.Concrete
	c.mu.Unlock()

	val, buildErr := c.build(concrete)

	c.mu.Lock()
	slot.val, slot.err = val, buildErr
	if buildErr == nil {
		c.instances[abstract] = val
	} else {
		delete(c.slots, abstract)
	}
	close(slot.ready)
	c.mu.Unlock()
	c.popResolving()
	return val, buildErr
}

// MustMake resolves a binding or panics.
func (c *Container) MustMake(abstract string) any {
	resolved, err := c.Make(abstract)
	if err != nil {
		panic(err)
	}
	return resolved
}

func (c *Container) resolveAlias(abstract string) (string, error) {
	seen := map[string]struct{}{}
	for {
		if _, ok := seen[abstract]; ok {
			return "", fmt.Errorf("container: alias cycle involving %q", abstract)
		}
		target, ok := c.aliases[abstract]
		if !ok {
			return abstract, nil
		}
		seen[abstract] = struct{}{}
		abstract = target
	}
}

func (c *Container) pushResolving(abstract string) error {
	id := goroutineID()
	raw, _ := c.chains.Load(id)
	var chain []string
	if raw != nil {
		chain = raw.([]string)
	}
	for _, prev := range chain {
		if prev == abstract {
			cycle := append(append([]string{}, chain...), abstract)
			return fmt.Errorf("container: circular dependency: %s", strings.Join(cycle, " -> "))
		}
	}
	next := make([]string, len(chain)+1)
	copy(next, chain)
	next[len(chain)] = abstract
	c.chains.Store(id, next)
	return nil
}

func (c *Container) popResolving() {
	id := goroutineID()
	raw, ok := c.chains.Load(id)
	if !ok {
		return
	}
	chain := raw.([]string)
	if len(chain) <= 1 {
		c.chains.Delete(id)
		return
	}
	next := make([]string, len(chain)-1)
	copy(next, chain[:len(chain)-1])
	c.chains.Store(id, next)
}

func (c *Container) build(concrete any) (any, error) {
	switch v := concrete.(type) {
	case func(*Container) any:
		return v(c), nil
	case func(*Container) (any, error):
		return v(c)
	case func() any:
		return v(), nil
	default:
		return c.buildReflect(concrete)
	}
}

func (c *Container) buildReflect(concrete any) (any, error) {
	rv := reflect.ValueOf(concrete)
	if rv.Kind() != reflect.Func {
		return concrete, nil
	}
	rt := rv.Type()
	if rt.NumIn() != 0 {
		return nil, fmt.Errorf("container: factory %s must have no parameters (use func(*container.Container) any)", rt)
	}
	if rt.NumOut() < 1 || rt.NumOut() > 2 {
		return nil, fmt.Errorf("container: factory %s must return T or (T, error)", rt)
	}
	if rt.NumOut() == 2 && !rt.Out(1).Implements(errorType) {
		return nil, fmt.Errorf("container: factory %s second return value must be error", rt)
	}
	results := rv.Call(nil)
	if rt.NumOut() == 2 && !results[1].IsNil() {
		if err, ok := results[1].Interface().(error); ok {
			return nil, err
		}
		return nil, fmt.Errorf("container: factory %s returned non-error second value", rt)
	}
	if !results[0].IsValid() {
		return nil, fmt.Errorf("container: factory returned no value")
	}
	if results[0].Kind() == reflect.Ptr || results[0].Kind() == reflect.Interface ||
		results[0].Kind() == reflect.Slice || results[0].Kind() == reflect.Map ||
		results[0].Kind() == reflect.Chan || results[0].Kind() == reflect.Func {
		if results[0].IsNil() {
			return nil, nil
		}
	}
	return results[0].Interface(), nil
}
