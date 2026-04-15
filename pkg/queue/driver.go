package queue

import (
	"context"
	"time"
)

// Driver is the backend interface for queue storage.
// The Redis implementation uses LPUSH/BRPOP for FIFO processing.
type Driver interface {
	// Push adds a payload to the end of the named queue.
	Push(ctx context.Context, queue string, payload []byte) error

	// Pop blocks until a payload is available on any of the named queues
	// and returns it. Returns nil payload when context is cancelled.
	Pop(ctx context.Context, queues ...string) ([]byte, error)

	// LaterAt schedules a payload to become available at the given time.
	LaterAt(ctx context.Context, queue string, at time.Time, payload []byte) error

	// MigrateDelayed moves delayed payloads whose scheduled time has passed
	// into the ready queue. Called periodically by the worker.
	MigrateDelayed(ctx context.Context, queue string) error

	// Size returns the number of pending payloads in the given queue.
	Size(ctx context.Context, queue string) (int64, error)

	// Flush removes all payloads from the given queue.
	Flush(ctx context.Context, queue string) error

	// Close releases resources.
	Close() error
}
