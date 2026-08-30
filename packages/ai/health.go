package ai

import (
	"context"
	"strings"
	"time"
)

// Healthy is an optional probe for provider liveness.
type Healthy interface {
	Health(ctx context.Context) error
}

// HealthStatus is the result of a provider health check.
type HealthStatus struct {
	Provider  string
	OK        bool
	Latency   time.Duration
	Err       error
	CheckedAt time.Time
	Skipped   bool // true when driver does not implement Healthy
}

// CheckHealth probes one or more providers (default provider if none named).
func (m *Manager) CheckHealth(ctx context.Context, providers ...string) []HealthStatus {
	if m == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	names := providers
	if len(names) == 0 {
		m.mu.RLock()
		def := m.defaultDriver
		m.mu.RUnlock()
		names = []string{def}
	}
	out := make([]HealthStatus, 0, len(names))
	for _, name := range names {
		out = append(out, m.checkOne(ctx, name))
	}
	return out
}

// CheckHealthAll probes every registered provider (sorted by name).
func (m *Manager) CheckHealthAll(ctx context.Context) []HealthStatus {
	if m == nil {
		return nil
	}
	return m.CheckHealth(ctx, m.Drivers()...)
}

func (m *Manager) checkOne(ctx context.Context, name string) HealthStatus {
	name = strings.ToLower(strings.TrimSpace(name))
	st := HealthStatus{Provider: name, CheckedAt: time.Now().UTC()}
	d, err := m.Driver(name)
	if err != nil {
		st.Err = err
		return st
	}
	h, ok := d.(Healthy)
	if !ok {
		st.Skipped = true
		st.OK = true // registered but no probe
		return st
	}
	start := time.Now()
	err = h.Health(ctx)
	st.Latency = time.Since(start)
	if err != nil {
		st.Err = err
		return st
	}
	st.OK = true
	return st
}

// HealthyProviders returns provider names that reported OK with no error.
func HealthyProviders(statuses []HealthStatus) []string {
	out := make([]string, 0, len(statuses))
	for _, s := range statuses {
		if s.OK && s.Err == nil {
			out = append(out, s.Provider)
		}
	}
	return out
}

// HealthError wraps a failed probe as KindUnavailable.
func HealthError(provider string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Kind: KindUnavailable, Provider: provider, Err: err}
}
