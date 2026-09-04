package config

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Repository stores nested configuration values.
type Repository struct {
	mu     sync.RWMutex
	values map[string]any
	frozen bool
}

// New creates an empty configuration repository.
func New() *Repository {
	return &Repository{values: make(map[string]any)}
}

// Freeze makes the repository read-only. Load/Set after freeze panic.
func (r *Repository) Freeze() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frozen = true
}

// Frozen reports whether writes are rejected.
func (r *Repository) Frozen() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.frozen
}

func (r *Repository) mustWritable() {
	if r.frozen {
		panic("config: repository is frozen after bootstrap")
	}
}

// Set stores a configuration value using dot notation.
func (r *Repository) Set(key string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mustWritable()
	r.setNested(r.values, strings.Split(key, "."), deepCopyValue(value))
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
	return deepCopyValue(current)
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

// All returns a recursive copy of maps and slices. Pointers and structs
// inside config values are not cloned.
func (r *Repository) All() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return deepCopyMap(r.values)
}

// Load merges a nested map into the repository under an optional prefix.
func (r *Repository) Load(name string, values map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mustWritable()
	if name == "" {
		for key, value := range values {
			r.values[key] = deepCopyValue(value)
		}
		return
	}
	r.values[name] = deepCopyMap(values)
}

// Loader is the config surface LoadIfAbsent needs.
type Loader interface {
	Get(key string, fallback ...any) any
	Load(name string, values map[string]any)
}

// LoadIfAbsent loads values under name only when that key is not already present
// (so a config cache snapshot is not overwritten with defaults).
func LoadIfAbsent(repo Loader, name string, values map[string]any) {
	if repo == nil || name == "" {
		return
	}
	if repo.Get(name) != nil {
		return
	}
	repo.Load(name, values)
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
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = deepCopyValue(value)
	}
	return dst
}

// deepCopyValue recursively copies maps and slices. Pointers, structs, and
// other reference types are returned as-is (framework config uses primitives
// and nested maps/slices).
func deepCopyValue(value any) any {
	if value == nil {
		return nil
	}
	if m, ok := value.(map[string]any); ok {
		return deepCopyMap(m)
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Map:
		if rv.IsNil() {
			return value
		}
		out := reflect.MakeMapWithSize(rv.Type(), rv.Len())
		iter := rv.MapRange()
		elem := rv.Type().Elem()
		for iter.Next() {
			copied := deepCopyValue(iter.Value().Interface())
			var val reflect.Value
			if copied == nil {
				val = reflect.Zero(elem)
			} else {
				val = reflect.ValueOf(copied)
			}
			out.SetMapIndex(iter.Key(), val)
		}
		return out.Interface()
	case reflect.Slice:
		if rv.IsNil() {
			return value
		}
		n := rv.Len()
		out := reflect.MakeSlice(rv.Type(), n, n)
		for i := 0; i < n; i++ {
			copied := deepCopyValue(rv.Index(i).Interface())
			if copied == nil {
				continue
			}
			cv := reflect.ValueOf(copied)
			if cv.IsValid() {
				out.Index(i).Set(cv)
			}
		}
		return out.Interface()
	default:
		return value
	}
}
