package repository

import (
	"sync"

	"github.com/zatrano/framework/v2/examples/reference/internal/domain"
)

// Memory is an in-memory Repository. A real consumer would bind a database
// package through enablement; this module cannot import
// github.com/zatrano/packages (architecture freeze).
type Memory struct {
	mu    sync.RWMutex
	items map[string]domain.Item
}

// NewMemory returns an empty store.
func NewMemory() *Memory {
	return &Memory{items: map[string]domain.Item{}}
}

// Seed inserts or replaces an item.
func (m *Memory) Seed(item domain.Item) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.items == nil {
		m.items = map[string]domain.Item{}
	}
	m.items[item.ID] = item
}

// Get implements domain.Repository.
func (m *Memory) Get(id string) (domain.Item, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.items[id]
	if !ok {
		return domain.Item{}, domain.ErrNotFound
	}
	return item, nil
}

// List implements domain.Repository.
func (m *Memory) List() []domain.Item {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.Item, 0, len(m.items))
	for _, item := range m.items {
		out = append(out, item)
	}
	return out
}

// Unavailable is a repository that always fails, wrapping a cause with
// domain.ErrUnavailable so HTTP can map the sentinel without exposing the cause.
type Unavailable struct {
	Cause error
}

// Get implements domain.Repository.
func (u *Unavailable) Get(id string) (domain.Item, error) {
	if u == nil || u.Cause == nil {
		return domain.Item{}, domain.ErrUnavailable
	}
	return domain.Item{}, wrapUnavailable(u.Cause)
}

// List implements domain.Repository.
func (u *Unavailable) List() []domain.Item {
	return nil
}

func wrapUnavailable(cause error) error {
	return unavailableError{cause: cause}
}

type unavailableError struct {
	cause error
}

func (e unavailableError) Error() string {
	if e.cause == nil {
		return domain.ErrUnavailable.Error()
	}
	return domain.ErrUnavailable.Error() + ": " + e.cause.Error()
}

func (e unavailableError) Unwrap() error {
	return e.cause
}

func (e unavailableError) Is(target error) bool {
	return target == domain.ErrUnavailable
}
