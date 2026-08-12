package events

// Subscriber registers one or more listeners on a dispatcher.
type Subscriber interface {
	Subscribe(d *Dispatcher)
}

// Register registers a subscriber's listeners.
func (d *Dispatcher) Register(sub Subscriber) {
	if d == nil || sub == nil {
		return
	}
	sub.Subscribe(d)
}

// RegisterSubscribers registers multiple subscribers.
func (d *Dispatcher) RegisterSubscribers(subs ...Subscriber) {
	for _, sub := range subs {
		d.Register(sub)
	}
}
