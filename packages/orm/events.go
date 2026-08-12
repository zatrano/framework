package orm

import (
	"reflect"
	"strings"
	"sync"

	"github.com/zatrano/framework/packages/events"
)

var (
	dispatcherMu sync.RWMutex
	dispatcher   *events.Dispatcher
)

// SetDispatcher wires a global event dispatcher for model lifecycle events.
func SetDispatcher(d *events.Dispatcher) {
	dispatcherMu.Lock()
	defer dispatcherMu.Unlock()
	dispatcher = d
}

// Dispatcher returns the configured model event dispatcher.
func Dispatcher() *events.Dispatcher {
	dispatcherMu.RLock()
	defer dispatcherMu.RUnlock()
	return dispatcher
}

func dispatchModel(action string, model any) error {
	d := Dispatcher()
	if d == nil || model == nil {
		return nil
	}
	subject := eventSubject(model)
	if subject == "" {
		return nil
	}
	return d.Dispatch(subject+"."+action, model)
}

func eventSubject(model any) string {
	rt := reflect.TypeOf(model)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	name := rt.Name()
	if name == "" {
		return ""
	}
	return strings.ToLower(name)
}
