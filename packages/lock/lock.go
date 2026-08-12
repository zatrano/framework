package lock

import (
	"fmt"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/support/uuid"
)

type entry struct {
	owner     string
	expiresAt time.Time
}

// Manager provides process-local named locks (cache-like API).
type Manager struct {
	mu    sync.Mutex
	items map[string]entry
}

// Lock is a named lock handle.
type Lock struct {
	manager *Manager
	name    string
	owner   string
	ttl     time.Duration
}

// New creates a lock manager.
func New() *Manager {
	return &Manager{items: make(map[string]entry)}
}

// Get returns a lock handle for name.
func (m *Manager) Get(name string, ttl ...time.Duration) *Lock {
	duration := 10 * time.Second
	if len(ttl) > 0 && ttl[0] > 0 {
		duration = ttl[0]
	}
	return &Lock{
		manager: m,
		name:    name,
		owner:   uuid.New(),
		ttl:     duration,
	}
}

// Acquire tries to obtain the lock.
func (l *Lock) Acquire() bool {
	l.manager.mu.Lock()
	defer l.manager.mu.Unlock()
	l.manager.purgeLocked()
	if e, ok := l.manager.items[l.name]; ok && time.Now().Before(e.expiresAt) {
		return false
	}
	l.manager.items[l.name] = entry{owner: l.owner, expiresAt: time.Now().Add(l.ttl)}
	return true
}

// Block waits until the lock is acquired or timeout elapses.
func (l *Lock) Block(wait time.Duration) bool {
	deadline := time.Now().Add(wait)
	for {
		if l.Acquire() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Release frees the lock if owned by this handle.
func (l *Lock) Release() bool {
	l.manager.mu.Lock()
	defer l.manager.mu.Unlock()
	e, ok := l.manager.items[l.name]
	if !ok || e.owner != l.owner {
		return false
	}
	delete(l.manager.items, l.name)
	return true
}

// Get is an alias for Acquire (framework familiarity).
func (l *Lock) Get() bool { return l.Acquire() }

// Run acquires a lock, runs fn, then releases.
func (m *Manager) Run(name string, ttl time.Duration, fn func() error) error {
	lock := m.Get(name, ttl)
	if !lock.Acquire() {
		return fmt.Errorf("lock: unable to acquire [%s]", name)
	}
	defer lock.Release()
	return fn()
}

func (m *Manager) purgeLocked() {
	now := time.Now()
	for k, e := range m.items {
		if now.After(e.expiresAt) {
			delete(m.items, k)
		}
	}
}
