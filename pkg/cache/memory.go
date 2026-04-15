package cache

import (
	"context"
	"sync"
	"time"
)

// memoryItem holds a cached value with an optional expiration.
type memoryItem struct {
	value     string
	expiresAt time.Time // zero means no expiration
}

func (i memoryItem) expired() bool {
	return !i.expiresAt.IsZero() && time.Now().After(i.expiresAt)
}

// MemoryDriver is a thread-safe, in-process cache backed by a Go map.
// Suitable for single-process deployments or tests. Items are lazily evicted on read.
type MemoryDriver struct {
	mu    sync.RWMutex
	items map[string]memoryItem
}

// NewMemoryDriver creates a ready-to-use in-memory cache driver.
func NewMemoryDriver() *MemoryDriver {
	return &MemoryDriver{items: make(map[string]memoryItem)}
}

// Ensure MemoryDriver implements Driver at compile time.
var _ Driver = (*MemoryDriver)(nil)

func (m *MemoryDriver) Get(_ context.Context, key string) (string, bool) {
	m.mu.RLock()
	item, ok := m.items[key]
	m.mu.RUnlock()
	if !ok {
		return "", false
	}
	if item.expired() {
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		return "", false
	}
	return item.value, true
}

func (m *MemoryDriver) Set(_ context.Context, key, value string, ttl time.Duration) error {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	m.mu.Lock()
	m.items[key] = memoryItem{value: value, expiresAt: exp}
	m.mu.Unlock()
	return nil
}

func (m *MemoryDriver) Has(_ context.Context, key string) bool {
	m.mu.RLock()
	item, ok := m.items[key]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	if item.expired() {
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		return false
	}
	return true
}

func (m *MemoryDriver) Delete(_ context.Context, keys ...string) error {
	m.mu.Lock()
	for _, k := range keys {
		delete(m.items, k)
	}
	m.mu.Unlock()
	return nil
}

func (m *MemoryDriver) TTL(_ context.Context, key string) (time.Duration, bool) {
	m.mu.RLock()
	item, ok := m.items[key]
	m.mu.RUnlock()
	if !ok || item.expired() {
		return 0, false
	}
	if item.expiresAt.IsZero() {
		return 0, true // exists, no expiration
	}
	return time.Until(item.expiresAt), true
}

func (m *MemoryDriver) Flush(_ context.Context) error {
	m.mu.Lock()
	m.items = make(map[string]memoryItem)
	m.mu.Unlock()
	return nil
}

func (m *MemoryDriver) Close() error {
	return m.Flush(context.Background())
}
