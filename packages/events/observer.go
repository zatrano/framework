package events

// ModelObserver handles common model lifecycle events.
type ModelObserver interface {
	Created(event any) error
	Updated(event any) error
	Deleted(event any) error
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
func (d *Dispatcher) ObserveModel(subject string, observer ModelObserver) {
	if observer == nil {
		return
	}
	d.Observe(subject, map[string]Listener{
		"created": observer.Created,
		"updated": observer.Updated,
		"deleted": observer.Deleted,
	})
}
