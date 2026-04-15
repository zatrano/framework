package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/zatrano/framework/pkg/queue"
)

// Dispatcher is the central event bus. It manages listener registration
// and event dispatching (synchronous and queue-backed asynchronous).
type Dispatcher struct {
	mu        sync.RWMutex
	listeners map[string][]Listener
	logger    *zap.Logger
	queue     *queue.Manager // optional, nil if queue unavailable
}

// New creates an event dispatcher.
func New(logger *zap.Logger) *Dispatcher {
	return &Dispatcher{
		listeners: make(map[string][]Listener),
		logger:    logger,
	}
}

// SetQueue enables async listener dispatch via the queue system.
// When set, listeners implementing ShouldQueue are dispatched as jobs.
func (d *Dispatcher) SetQueue(q *queue.Manager) {
	d.queue = q
}

// ─── Registration ──────────────────────────────────────────────────────────

// Listen registers a listener for the given event name.
func (d *Dispatcher) Listen(eventName string, listener Listener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners[eventName] = append(d.listeners[eventName], listener)
}

// Subscribe registers multiple listeners for an event name.
func (d *Dispatcher) Subscribe(eventName string, listeners ...Listener) {
	for _, l := range listeners {
		d.Listen(eventName, l)
	}
}

// ListenFunc registers a function as a listener.
func (d *Dispatcher) ListenFunc(eventName string, fn func(ctx context.Context, event Event) error) {
	d.Listen(eventName, ListenerFunc(fn))
}

// ─── Dispatching ───────────────────────────────────────────────────────────

// Fire dispatches an event to all registered listeners.
// Synchronous listeners run inline; ShouldQueue listeners are dispatched to the queue.
// Returns the first error from a synchronous listener (other listeners still run).
func (d *Dispatcher) Fire(ctx context.Context, event Event) error {
	d.mu.RLock()
	listeners := d.listeners[event.Name()]
	d.mu.RUnlock()

	if len(listeners) == 0 {
		return nil
	}

	d.logger.Debug("event fired",
		zap.String("event", event.Name()),
		zap.Int("listeners", len(listeners)),
	)

	var firstErr error
	for _, l := range listeners {
		if sq, ok := l.(ShouldQueue); ok {
			if err := d.dispatchAsync(ctx, event, sq); err != nil {
				d.logger.Error("event queue dispatch failed",
					zap.String("event", event.Name()),
					zap.Error(err),
				)
				if firstErr == nil {
					firstErr = err
				}
			}
			continue
		}

		// Synchronous dispatch with panic recovery.
		if err := d.dispatchSync(ctx, event, l); err != nil {
			d.logger.Error("event listener failed",
				zap.String("event", event.Name()),
				zap.Error(err),
			)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// FireAsync dispatches an event to all listeners asynchronously (goroutines).
// Errors are only logged, not returned.
func (d *Dispatcher) FireAsync(ctx context.Context, event Event) {
	d.mu.RLock()
	listeners := d.listeners[event.Name()]
	d.mu.RUnlock()

	for _, l := range listeners {
		go func(listener Listener) {
			if sq, ok := listener.(ShouldQueue); ok {
				_ = d.dispatchAsync(ctx, event, sq)
				return
			}
			_ = d.dispatchSync(ctx, event, listener)
		}(l)
	}
}

// HasListeners returns true if the event has at least one listener.
func (d *Dispatcher) HasListeners(eventName string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.listeners[eventName]) > 0
}

// ─── Internal ──────────────────────────────────────────────────────────────

func (d *Dispatcher) dispatchSync(ctx context.Context, event Event, listener Listener) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("event listener panic: %v", r)
			d.logger.Error("event listener panic",
				zap.String("event", event.Name()),
				zap.Any("panic", r),
			)
		}
	}()
	return listener.Handle(ctx, event)
}

func (d *Dispatcher) dispatchAsync(ctx context.Context, event Event, sq ShouldQueue) error {
	if d.queue == nil {
		// Fallback: run synchronously if no queue configured.
		d.logger.Warn("ShouldQueue listener running synchronously (no queue configured)",
			zap.String("event", event.Name()),
		)
		return d.dispatchSync(ctx, event, sq)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("event marshal: %w", err)
	}

	job := &EventListenerJob{
		EventName:    event.Name(),
		EventPayload: payload,
		MaxRetries:   sq.Retries(),
		QueueName:    sq.Queue(),
	}
	if job.QueueName == "" {
		job.QueueName = "events"
	}
	if job.MaxRetries <= 0 {
		job.MaxRetries = 3
	}

	return d.queue.Dispatch(ctx, job)
}

// ─── EventListenerJob ──────────────────────────────────────────────────────

// EventListenerJob is a queue job that replays an event to its listeners.
type EventListenerJob struct {
	queue.BaseJob
	EventName    string          `json:"event_name"`
	EventPayload json.RawMessage `json:"event_payload"`
	MaxRetries   int             `json:"max_retries"`
	QueueName    string          `json:"queue_name"`
}

func (j *EventListenerJob) Name() string            { return "zatrano_event_listener" }
func (j *EventListenerJob) Queue() string           { return j.QueueName }
func (j *EventListenerJob) Retries() int            { return j.MaxRetries }
func (j *EventListenerJob) Timeout() time.Duration  { return 60 * time.Second }

func (j *EventListenerJob) Handle(_ context.Context) error {
	return fmt.Errorf("event listener job requires dispatcher registration — use events.RegisterEventJob()")
}

// RegisterEventJob registers the event listener job with the queue manager.
func RegisterEventJob(qm *queue.Manager, d *Dispatcher) {
	qm.Register("zatrano_event_listener", func() queue.Job {
		return &eventListenerJobWithDispatcher{dispatcher: d}
	})
}

type eventListenerJobWithDispatcher struct {
	queue.BaseJob
	EventName    string          `json:"event_name"`
	EventPayload json.RawMessage `json:"event_payload"`
	MaxRetries   int             `json:"max_retries"`
	QueueName    string          `json:"queue_name"`
	dispatcher   *Dispatcher
}

func (j *eventListenerJobWithDispatcher) Name() string            { return "zatrano_event_listener" }
func (j *eventListenerJobWithDispatcher) Queue() string           { return j.QueueName }
func (j *eventListenerJobWithDispatcher) Retries() int            { return j.MaxRetries }
func (j *eventListenerJobWithDispatcher) Timeout() time.Duration  { return 60 * time.Second }

func (j *eventListenerJobWithDispatcher) Handle(ctx context.Context) error {
	// Create a raw event wrapper and re-fire synchronously.
	raw := &rawEvent{
		name:    j.EventName,
		payload: j.EventPayload,
	}
	return j.dispatcher.Fire(ctx, raw)
}

// rawEvent wraps deserialized event data for replay.
type rawEvent struct {
	name    string
	payload json.RawMessage
}

func (e *rawEvent) Name() string { return e.name }
