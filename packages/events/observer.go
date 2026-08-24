package events

// ModelObserver handles common model lifecycle events.
type ModelObserver interface {
	Created(event any) error
	Updated(event any) error
	Deleted(event any) error
}

// LifecycleObserver extends ModelObserver with saving/saved/retrieved/replicating/restoring hooks.
type LifecycleObserver interface {
	ModelObserver
	Saving(event any) error
	Saved(event any) error
	Retrieved(event any) error
	Replicating(event any) error
	Restoring(event any) error
	Restored(event any) error
	ForceDeleted(event any) error
}

// Observe registers handlers under "{subject}.{action}" event names.
func (d *Dispatcher) Observe(subject string, handlers map[string]Listener) {
	for action, listener := range handlers {
		if listener == nil {
			continue
		}
		d.Listen(subject+"."+action, listener)
	}
}

// ObserveModel registers created/updated/deleted listeners for a subject.
// When observer also implements LifecycleObserver, extra lifecycle events are wired.
func (d *Dispatcher) ObserveModel(subject string, observer ModelObserver) {
	if observer == nil {
		return
	}
	handlers := map[string]Listener{
		"created": observer.Created,
		"updated": observer.Updated,
		"deleted": observer.Deleted,
	}
	if life, ok := observer.(LifecycleObserver); ok {
		handlers["saving"] = life.Saving
		handlers["saved"] = life.Saved
		handlers["retrieved"] = life.Retrieved
		handlers["replicating"] = life.Replicating
		handlers["restoring"] = life.Restoring
		handlers["restored"] = life.Restored
		handlers["forceDeleted"] = life.ForceDeleted
	}
	d.Observe(subject, handlers)
}
