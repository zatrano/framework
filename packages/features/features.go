package features

import (
	"sync"
)

// Context can influence flag evaluation (user, tenant, env).
type Context map[string]any

// Gate decides whether a feature is enabled for a context.
type Gate func(ctx Context) bool

// Manager stores feature flags and percentage rollouts.
type Manager struct {
	mu      sync.RWMutex
	flags   map[string]bool
	gates   map[string]Gate
	rollout map[string]int // 0-100
}

// New creates a feature flag manager.
func New() *Manager {
	return &Manager{
		flags:   map[string]bool{},
		gates:   map[string]Gate{},
		rollout: map[string]int{},
	}
}

// Activate enables a feature globally.
func (m *Manager) Activate(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flags[name] = true
	delete(m.gates, name)
	delete(m.rollout, name)
}

// Deactivate disables a feature globally.
func (m *Manager) Deactivate(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flags[name] = false
	delete(m.gates, name)
	delete(m.rollout, name)
}

// Set sets a boolean flag value.
func (m *Manager) Set(name string, enabled bool) {
	if enabled {
		m.Activate(name)
		return
	}
	m.Deactivate(name)
}

// Define registers a custom gate for a feature.
func (m *Manager) Define(name string, gate Gate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gates[name] = gate
	delete(m.flags, name)
	delete(m.rollout, name)
}

// Rollout enables a feature for a percentage of subjects (0-100).
func (m *Manager) Rollout(name string, percent int) {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rollout[name] = percent
	delete(m.flags, name)
	delete(m.gates, name)
}

// Active reports whether a feature is enabled for an optional context.
func (m *Manager) Active(name string, ctx ...Context) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if gate, ok := m.gates[name]; ok {
		c := Context{}
		if len(ctx) > 0 && ctx[0] != nil {
			c = ctx[0]
		}
		return gate(c)
	}
	if percent, ok := m.rollout[name]; ok {
		c := Context{}
		if len(ctx) > 0 && ctx[0] != nil {
			c = ctx[0]
		}
		key := stringify(c["key"])
		if key == "" {
			key = stringify(c["user"])
		}
		if key == "" {
			key = name
		}
		return hashPercent(key+":"+name) < percent
	}
	if enabled, ok := m.flags[name]; ok {
		return enabled
	}
	return false
}

// Inactive is the inverse of Active.
func (m *Manager) Inactive(name string, ctx ...Context) bool {
	return !m.Active(name, ctx...)
}

// All returns a snapshot of known flags and their effective global state.
func (m *Manager) All(ctx ...Context) map[string]bool {
	m.mu.RLock()
	names := make(map[string]struct{})
	for name := range m.flags {
		names[name] = struct{}{}
	}
	for name := range m.gates {
		names[name] = struct{}{}
	}
	for name := range m.rollout {
		names[name] = struct{}{}
	}
	m.mu.RUnlock()

	out := make(map[string]bool, len(names))
	for name := range names {
		out[name] = m.Active(name, ctx...)
	}
	return out
}

func stringify(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func hashPercent(input string) int {
	var h uint32 = 2166136261
	for i := 0; i < len(input); i++ {
		h ^= uint32(input[i])
		h *= 16777619
	}
	return int(h % 100)
}
