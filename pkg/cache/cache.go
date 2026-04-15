package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Manager is the high-level cache facade. It wraps a Driver and adds
// convenience methods like Remember and JSON helpers.
type Manager struct {
	driver Driver
}

// New creates a cache manager wrapping the given driver.
func New(driver Driver) *Manager {
	return &Manager{driver: driver}
}

// Driver returns the underlying cache driver.
func (m *Manager) Driver() Driver { return m.driver }

// ─── Passthrough methods ───────────────────────────────────────────────────

// Get retrieves a raw string value. Returns ("", false) on miss.
func (m *Manager) Get(ctx context.Context, key string) (string, bool) {
	return m.driver.Get(ctx, key)
}

// Set stores a raw string value with optional TTL. Use ttl=0 for no expiration.
func (m *Manager) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return m.driver.Set(ctx, key, value, ttl)
}

// Has returns true if the key exists.
func (m *Manager) Has(ctx context.Context, key string) bool {
	return m.driver.Has(ctx, key)
}

// Delete removes one or more keys.
func (m *Manager) Delete(ctx context.Context, keys ...string) error {
	return m.driver.Delete(ctx, keys...)
}

// TTL returns the remaining time-to-live for a key.
func (m *Manager) TTL(ctx context.Context, key string) (time.Duration, bool) {
	return m.driver.TTL(ctx, key)
}

// Flush removes all entries from the cache.
func (m *Manager) Flush(ctx context.Context) error {
	return m.driver.Flush(ctx)
}

// Close releases driver resources.
func (m *Manager) Close() error {
	return m.driver.Close()
}

// ─── JSON helpers ──────────────────────────────────────────────────────────

// GetJSON retrieves the value for key and JSON-decodes it into dest.
// Returns false on cache miss.
func (m *Manager) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	raw, ok := m.driver.Get(ctx, key)
	if !ok {
		return false, nil
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return false, fmt.Errorf("cache: json decode %q: %w", key, err)
	}
	return true, nil
}

// SetJSON JSON-encodes value and stores it.
func (m *Manager) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: json encode %q: %w", key, err)
	}
	return m.driver.Set(ctx, key, string(data), ttl)
}

// ─── Remember ──────────────────────────────────────────────────────────────

// Remember returns the cached value for key. On cache miss, it calls fn to
// compute the value, stores the result with the given TTL, and returns it.
// This is the most commonly used cache pattern (Laravel's Cache::remember).
//
// Usage:
//
//	val, err := cache.Remember(ctx, "users:count", 5*time.Minute, func() (string, error) {
//	    count, err := db.CountUsers(ctx)
//	    return strconv.Itoa(count), err
//	})
func (m *Manager) Remember(ctx context.Context, key string, ttl time.Duration, fn func() (string, error)) (string, error) {
	if val, ok := m.driver.Get(ctx, key); ok {
		return val, nil
	}
	val, err := fn()
	if err != nil {
		return "", err
	}
	if err := m.driver.Set(ctx, key, val, ttl); err != nil {
		return val, fmt.Errorf("cache: remember set %q: %w", key, err)
	}
	return val, nil
}

// RememberJSON is like Remember but with automatic JSON marshal/unmarshal.
// The dest parameter receives the cached or freshly computed value.
//
// Usage:
//
//	var users []User
//	err := cache.RememberJSON(ctx, "users:list", 10*time.Minute, &users, func() (any, error) {
//	    return db.FindAllUsers(ctx)
//	})
func (m *Manager) RememberJSON(ctx context.Context, key string, ttl time.Duration, dest any, fn func() (any, error)) error {
	raw, ok := m.driver.Get(ctx, key)
	if ok {
		return json.Unmarshal([]byte(raw), dest)
	}
	val, err := fn()
	if err != nil {
		return err
	}
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("cache: remember json encode %q: %w", key, err)
	}
	if err := m.driver.Set(ctx, key, string(data), ttl); err != nil {
		return fmt.Errorf("cache: remember set %q: %w", key, err)
	}
	return json.Unmarshal(data, dest)
}

// ─── Forever ───────────────────────────────────────────────────────────────

// Forever stores a value with no expiration.
func (m *Manager) Forever(ctx context.Context, key, value string) error {
	return m.driver.Set(ctx, key, value, 0)
}

// ForeverJSON stores a JSON-encoded value with no expiration.
func (m *Manager) ForeverJSON(ctx context.Context, key string, value any) error {
	return m.SetJSON(ctx, key, value, 0)
}

// ─── Pull ──────────────────────────────────────────────────────────────────

// Pull retrieves and deletes a value in one call.
func (m *Manager) Pull(ctx context.Context, key string) (string, bool) {
	val, ok := m.driver.Get(ctx, key)
	if ok {
		_ = m.driver.Delete(ctx, key)
	}
	return val, ok
}

// ─── Increment / Decrement (string-based) ──────────────────────────────────

// Add stores a value only if the key does not already exist. Returns false if key exists.
func (m *Manager) Add(ctx context.Context, key, value string, ttl time.Duration) bool {
	if m.driver.Has(ctx, key) {
		return false
	}
	_ = m.driver.Set(ctx, key, value, ttl)
	return true
}

// ─── Tags ──────────────────────────────────────────────────────────────────

// Tags returns a TaggedCache that tracks keys under the given tag names.
// Tag-based invalidation is only supported with the Redis driver.
//
// Usage:
//
//	cache.Tags("users").Set(ctx, "users:1", data, 10*time.Minute)
//	cache.Tags("users", "posts").Flush(ctx) // invalidates all keys under both tags
func (m *Manager) Tags(tags ...string) *TaggedCache {
	return &TaggedCache{manager: m, tags: tags}
}
