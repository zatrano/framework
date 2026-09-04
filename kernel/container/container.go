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
	ready     chan struct{}
	val       any
	err       error
	waitingOn string
}

// Container is the ZATRANO factory-based service container.
//
// Nested Make uses an explicit resolution view (stack on a child *Container)
// so cycle detection does not parse goroutine IDs. Shared singleton cycles
// across goroutines use the slot wait-graph.
type Container struct {
	mu        sync.Mutex
	bindings  map[string]Binding
	instances map[string]any
	aliases   map[string]string
	slots     map[string]*sharedSlot
	frozen    bool
	root      *Container
	stack     []string
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

func (c *Container) core() *Container {
	if c == nil {
		return nil
	}
	if c.root != nil {
		return c.root
	}
	return c
}

func (c *Container) view(stack []string) *Container {
	return &Container{root: c.core(), stack: stack}
}

// Bind registers a non-shared binding.
func (c *Container) Bind(abstract string, concrete any) {
	c = c.core()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mustMutate()
	delete(c.instances, abstract)
	delete(c.slots, abstract)
	c.bindings[abstract] = Binding{Concrete: concrete, Shared: false}
}

// Singleton registers a shared binding.
func (c *Container) Singleton(abstract string, concrete any) {
	c = c.core()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mustMutate()
	delete(c.instances, abstract)
	delete(c.slots, abstract)
	c.bindings[abstract] = Binding{Concrete: concrete, Shared: true}
}

// Instance registers an existing shared instance.
func (c *Container) Instance(abstract string, instance any) {
	c = c.core()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mustMutate()
	delete(c.slots, abstract)
	c.instances[abstract] = instance
	c.bindings[abstract] = Binding{Concrete: instance, Shared: true}
}

// Alias creates an alias that resolves to abstract.
func (c *Container) Alias(abstract, alias string) {
	c = c.core()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mustMutate()
	c.aliases[alias] = abstract
}

// Freeze locks registration (Bind/Singleton/Instance/Alias). Make still
// publishes lazy singleton instances: frozen means the binding graph is
// immutable, not that the instance map is frozen.
func (c *Container) Freeze() {
	c = c.core()
	if c == nil {
		return
	}
	c.mu.Lock()
	c.frozen = true
	c.mu.Unlock()
}

// Frozen reports whether registration is locked.
func (c *Container) Frozen() bool {
	c = c.core()
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.frozen
}

func (c *Container) mustMutate() {
	if c.frozen {
		panic("container: cannot register bindings after bootstrap")
	}
}

// Bound reports whether an abstract is bound.
func (c *Container) Bound(abstract string) bool {
	c = c.core()
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
	if c == nil {
		return nil, fmt.Errorf("container: nil container")
	}
	stack := c.stack
	return c.core().resolve(stack, abstract)
}

func (c *Container) resolve(stack []string, abstract string) (any, error) {
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

	if err := cycleIn(stack, abstract); err != nil {
		c.mu.Unlock()
		return nil, err
	}
	next := appendStack(stack, abstract)
	parent := ""
	if len(stack) > 0 {
		parent = stack[len(stack)-1]
	}

	if !binding.Shared {
		concrete := binding.Concrete
		c.mu.Unlock()
		return c.build(c.view(next), concrete)
	}

	if slot, ok := c.slots[abstract]; ok {
		if err := c.markWaitingLocked(parent, abstract); err != nil {
			c.mu.Unlock()
			return nil, err
		}
		c.mu.Unlock()
		<-slot.ready
		c.mu.Lock()
		c.clearWaitingLocked(parent)
		val, slotErr := slot.val, slot.err
		c.mu.Unlock()
		return val, slotErr
	}

	slot := &sharedSlot{ready: make(chan struct{})}
	c.slots[abstract] = slot
	if err := c.markWaitingLocked(parent, abstract); err != nil {
		delete(c.slots, abstract)
		c.mu.Unlock()
		return nil, err
	}
	concrete := binding.Concrete
	c.mu.Unlock()

	val, buildErr := c.build(c.view(next), concrete)

	c.mu.Lock()
	c.clearWaitingLocked(parent)
	slot.val, slot.err = val, buildErr
	if buildErr == nil {
		c.instances[abstract] = val
	} else {
		delete(c.slots, abstract)
	}
	close(slot.ready)
	c.mu.Unlock()
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

func cycleIn(stack []string, abstract string) error {
	for _, prev := range stack {
		if prev == abstract {
			cycle := appendStack(stack, abstract)
			return fmt.Errorf("container: circular dependency: %s", strings.Join(cycle, " -> "))
		}
	}
	return nil
}

func appendStack(stack []string, abstract string) []string {
	next := make([]string, len(stack)+1)
	copy(next, stack)
	next[len(stack)] = abstract
	return next
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

func (c *Container) markWaitingLocked(parent, target string) error {
	if parent == "" {
		return nil
	}
	ps, ok := c.slots[parent]
	if !ok {
		return nil
	}
	ps.waitingOn = target
	if c.waitCycleLocked(target) {
		ps.waitingOn = ""
		return fmt.Errorf("container: circular dependency: %s -> %s", parent, target)
	}
	return nil
}

func (c *Container) clearWaitingLocked(parent string) {
	if parent == "" {
		return
	}
	if ps, ok := c.slots[parent]; ok {
		ps.waitingOn = ""
	}
}

func (c *Container) waitCycleLocked(start string) bool {
	seen := map[string]struct{}{}
	cur := start
	for cur != "" {
		if _, ok := seen[cur]; ok {
			return true
		}
		seen[cur] = struct{}{}
		slot, ok := c.slots[cur]
		if !ok {
			return false
		}
		cur = slot.waitingOn
	}
	return false
}

func (c *Container) build(view *Container, concrete any) (any, error) {
	switch v := concrete.(type) {
	case func(*Container) any:
		return v(view), nil
	case func(*Container) (any, error):
		return v(view)
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
