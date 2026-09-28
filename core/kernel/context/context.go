package context

import (
	"sync"
)

// Store is a concurrent key/value context bag.
type Store struct {
	mu     sync.RWMutex
	values map[string]any
}

// New creates an empty context store.
func New() *Store {
	return &Store{values: make(map[string]any)}
}

// Add adds a value if the key does not exist.
func (s *Store) Add(key string, value any) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.values[key]; ok {
		return false
	}
	s.values[key] = value
	return true
}

// Put sets a value.
func (s *Store) Put(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
}

// Get returns a value.
func (s *Store) Get(key string, fallback ...any) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if value, ok := s.values[key]; ok {
		return value
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return nil
}

// Has reports whether a key exists.
func (s *Store) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.values[key]
	return ok
}

// Pull gets and forgets a value.
func (s *Store) Pull(key string, fallback ...any) any {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	if ok {
		delete(s.values, key)
		return value
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return nil
}

// Forget removes keys.
func (s *Store) Forget(keys ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, key := range keys {
		delete(s.values, key)
	}
}

// Flush clears all values.
func (s *Store) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = make(map[string]any)
}

// All returns a shallow copy of all values.
func (s *Store) All() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]any, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out
}

// Only returns a subset of values.
func (s *Store) Only(keys ...string) map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]any, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out
}
