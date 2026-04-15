package cache

import (
	"context"
	"time"
)

// Driver is the common interface for all cache backends (memory, Redis, etc.).
// Every method accepts a context for cancellation and timeout propagation.
type Driver interface {
	// Get retrieves a value by key. Returns ("", false) on miss.
	Get(ctx context.Context, key string) (string, bool)

	// Set stores a value with an expiration. Use ttl=0 for no expiration.
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// Has returns true if the key exists and has not expired.
	Has(ctx context.Context, key string) bool

	// Delete removes one or more keys.
	Delete(ctx context.Context, keys ...string) error

	// TTL returns the remaining time-to-live for a key.
	// Returns (0, false) if the key does not exist.
	TTL(ctx context.Context, key string) (time.Duration, bool)

	// Flush removes all keys managed by this driver.
	Flush(ctx context.Context) error

	// Close releases resources held by the driver.
	Close() error
}
