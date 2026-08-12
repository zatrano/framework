package cache

import (
	"fmt"
	"strconv"
	"time"
)

// Add stores a value only if the key does not already exist.
func Add(store Store, key string, value any, ttl time.Duration) bool {
	if store == nil || store.Has(key) {
		return false
	}
	return store.Put(key, value, ttl) == nil
}

// RememberForever caches the callback result indefinitely.
func RememberForever(store Store, key string, callback func() (any, error)) (any, error) {
	if store == nil {
		return callback()
	}
	if value, ok := store.Get(key); ok {
		return value, nil
	}
	value, err := callback()
	if err != nil {
		return nil, err
	}
	if err := store.Forever(key, value); err != nil {
		return nil, err
	}
	return value, nil
}

// Many returns multiple keys at once.
func Many(store Store, keys ...string) map[string]any {
	out := make(map[string]any, len(keys))
	if store == nil {
		return out
	}
	for _, key := range keys {
		if value, ok := store.Get(key); ok {
			out[key] = value
		}
	}
	return out
}

// PutMany stores multiple values with the same TTL.
func PutMany(store Store, values map[string]any, ttl time.Duration) error {
	if store == nil {
		return fmt.Errorf("cache store is nil")
	}
	for key, value := range values {
		if err := store.Put(key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

// Increment increases a numeric cache value.
func Increment(store Store, key string, by ...int64) (int64, error) {
	step := int64(1)
	if len(by) > 0 {
		step = by[0]
	}
	return bump(store, key, step)
}

// Decrement decreases a numeric cache value.
func Decrement(store Store, key string, by ...int64) (int64, error) {
	step := int64(1)
	if len(by) > 0 {
		step = by[0]
	}
	return bump(store, key, -step)
}

func bump(store Store, key string, step int64) (int64, error) {
	if store == nil {
		return 0, fmt.Errorf("cache store is nil")
	}
	current := int64(0)
	if raw, ok := store.Get(key); ok {
		current = toInt64(raw)
	}
	next := current + step
	if err := store.Forever(key, next); err != nil {
		return 0, err
	}
	return next, nil
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	case string:
		parsed, _ := strconv.ParseInt(n, 10, 64)
		return parsed
	default:
		parsed, _ := strconv.ParseInt(fmt.Sprint(v), 10, 64)
		return parsed
	}
}

// Add proxies to the default store.
func (m *Manager) Add(key string, value any, ttl time.Duration) bool {
	return Add(m.Store(), key, value, ttl)
}

// RememberForever proxies to the default store.
func (m *Manager) RememberForever(key string, callback func() (any, error)) (any, error) {
	return RememberForever(m.Store(), key, callback)
}

// Many proxies to the default store.
func (m *Manager) Many(keys ...string) map[string]any {
	return Many(m.Store(), keys...)
}

// PutMany proxies to the default store.
func (m *Manager) PutMany(values map[string]any, ttl time.Duration) error {
	return PutMany(m.Store(), values, ttl)
}

// Increment proxies to the default store.
func (m *Manager) Increment(key string, by ...int64) (int64, error) {
	return Increment(m.Store(), key, by...)
}

// Decrement proxies to the default store.
func (m *Manager) Decrement(key string, by ...int64) (int64, error) {
	return Decrement(m.Store(), key, by...)
}
