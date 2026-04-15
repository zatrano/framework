package events

import "context"

// Listener processes an event. Implement Handle to react to dispatched events.
type Listener interface {
	// Handle processes the event. Return an error to signal failure.
	Handle(ctx context.Context, event Event) error
}

// ShouldQueue marks a listener for asynchronous processing via the queue system.
// When a listener implements ShouldQueue, it will be dispatched as a job
// instead of running synchronously.
type ShouldQueue interface {
	Listener

	// Queue returns the queue name to dispatch to (e.g. "events", "notifications").
	// Return "" to use the default queue.
	Queue() string

	// Retries returns the maximum number of retry attempts for the queued job.
	Retries() int
}

// ListenerFunc is an adapter to use ordinary functions as Listener.
//
//	dispatcher.Listen("user.created", events.ListenerFunc(func(ctx context.Context, e events.Event) error {
//	    log.Println("user created!", e)
//	    return nil
//	}))
type ListenerFunc func(ctx context.Context, event Event) error

func (f ListenerFunc) Handle(ctx context.Context, event Event) error {
	return f(ctx, event)
}
