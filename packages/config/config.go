package config

import (
	"fmt"
	"strings"
	"sync"
)

// Repository stores nested configuration values.
type Repository struct {
	mu     sync.RWMutex
	values map[string]any
}

// New creates an empty configuration repository.
func New() *Repository {
	return &Repository{values: make(map[string]any)}
}

// Set stores a configuration value using dot notation.
func (r *Repository) Set(key string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.setNested(r.values, strings.Split(key, "."), value)
}

// Get retrieves a configuration value using dot notation.
func (r *Repository) Get(key string, fallback ...any) any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	current := any(r.values)
	for _, segment := range strings.Split(key, ".") {
		asMap, ok := current.(map[string]any)
		if !ok {
			if len(fallback) > 0 {
				return fallback[0]
			}
			return nil
		}
		next, exists := asMap[segment]
		if !exists {
			if len(fallback) > 0 {
				return fallback[0]
			}
			return nil
		}
		current = next
	}
	return current
}

// GetString returns a string configuration value.
func (r *Repository) GetString(key string, fallback ...string) string {
	value := r.Get(key)
	if value == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

// GetBool returns a bool configuration value.
func (r *Repository) GetBool(key string, fallback ...bool) bool {
	value := r.Get(key)
	if value == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1" || v == "yes" || v == "on"
	default:
		if len(fallback) > 0 {
			return fallback[0]
		}
		return false
	}
}

// GetInt returns an int configuration value.
func (r *Repository) GetInt(key string, fallback ...int) int {
	value := r.Get(key)
	if value == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
}

// All returns a copy of all configuration values.
func (r *Repository) All() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return deepCopyMap(r.values)
}

// Load merges a nested map into the repository under an optional prefix.
func (r *Repository) Load(name string, values map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if name == "" {
		for key, value := range values {
			r.values[key] = value
		}
		return
	}
	r.values[name] = values
}

func (r *Repository) setNested(root map[string]any, segments []string, value any) {
	if len(segments) == 1 {
		root[segments[0]] = value
		return
	}

	next, ok := root[segments[0]].(map[string]any)
	if !ok {
		next = make(map[string]any)
		root[segments[0]] = next
	}
	r.setNested(next, segments[1:], value)
}

func deepCopyMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for key, value := range src {
		if nested, ok := value.(map[string]any); ok {
			dst[key] = deepCopyMap(nested)
			continue
		}
		dst[key] = value
	}
	return dst
}
