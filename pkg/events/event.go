package events

import "time"

// Event is the interface that all application events must implement.
// Events carry data about something that happened in the application.
type Event interface {
	// Name returns the unique event name (e.g. "user.created", "order.placed").
	Name() string
}

// BaseEvent provides common fields for events.
type BaseEvent struct {
	// OccurredAt is when the event was fired.
	OccurredAt time.Time `json:"occurred_at"`
}

// NewBaseEvent creates a BaseEvent with the current timestamp.
func NewBaseEvent() BaseEvent {
	return BaseEvent{OccurredAt: time.Now()}
}
