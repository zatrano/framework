package events

import "sync"

// Listener handles an event payload.
type Listener func(event any) error

// Dispatcher dispatches events to listeners.
type Dispatcher struct {
	mu        sync.RWMutex
	listeners map[string][]Listener
}

// New creates an event dispatcher.
func New() *Dispatcher {
	return &Dispatcher{listeners: make(map[string][]Listener)}
}

// Listen registers a listener for an event name.
func (d *Dispatcher) Listen(name string, listener Listener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners[name] = append(d.listeners[name], listener)
}

// Dispatch fires an event by name.
func (d *Dispatcher) Dispatch(name string, event any) error {
	d.mu.RLock()
	listeners := append([]Listener{}, d.listeners[name]...)
	d.mu.RUnlock()

	for _, listener := range listeners {
		if err := listener(event); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe is an alias for Listen.
func (d *Dispatcher) Subscribe(name string, listener Listener) {
	d.Listen(name, listener)
}

// Forget removes listeners for an event.
func (d *Dispatcher) Forget(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.listeners, name)
}

// HasListeners reports whether an event has listeners.
func (d *Dispatcher) HasListeners(name string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.listeners[name]) > 0
}
