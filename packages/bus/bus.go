package bus

import (
	"fmt"
	"reflect"
	"sync"
)

// Handler handles a command and returns a result.
type Handler func(command any) (any, error)

// Bus dispatches commands to registered handlers.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string]Handler
	pipes    []func(command any, next func(any) (any, error)) (any, error)
}

// New creates a command bus.
func New() *Bus {
	return &Bus{handlers: make(map[string]Handler)}
}

// Map registers a handler for a command type or named key.
func (b *Bus) Map(command any, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[keyFor(command)] = handler
}

// MapNamed registers a handler under an explicit name.
func (b *Bus) MapNamed(name string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = handler
}

// Pipe adds a middleware around dispatch.
func (b *Bus) Pipe(pipe func(command any, next func(any) (any, error)) (any, error)) {
	b.pipes = append(b.pipes, pipe)
}

// Dispatch sends a command to its handler.
func (b *Bus) Dispatch(command any) (any, error) {
	key := keyFor(command)
	b.mu.RLock()
	handler, ok := b.handlers[key]
	b.mu.RUnlock()
	if !ok {
		// Try named string commands.
		if name, isString := command.(string); isString {
			b.mu.RLock()
			handler, ok = b.handlers[name]
			b.mu.RUnlock()
		}
	}
	if !ok {
		return nil, fmt.Errorf("no bus handler for [%s]", key)
	}

	next := func(cmd any) (any, error) {
		return handler(cmd)
	}
	for i := len(b.pipes) - 1; i >= 0; i-- {
		pipe := b.pipes[i]
		inner := next
		next = func(cmd any) (any, error) {
			return pipe(cmd, inner)
		}
	}
	return next(command)
}

// DispatchNamed dispatches using an explicit handler name and payload.
func (b *Bus) DispatchNamed(name string, payload any) (any, error) {
	b.mu.RLock()
	handler, ok := b.handlers[name]
	b.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no bus handler for [%s]", name)
	}
	return handler(payload)
}

// Has reports whether a handler exists.
func (b *Bus) Has(command any) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.handlers[keyFor(command)]
	return ok
}

func keyFor(command any) string {
	if command == nil {
		return "<nil>"
	}
	if name, ok := command.(string); ok {
		return name
	}
	t := reflect.TypeOf(command)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Name() != "" {
		return t.PkgPath() + "." + t.Name()
	}
	return t.String()
}
