package ai

import (
	"context"
	"fmt"
	"strings"
)

// LiveOption configures ProfileLive / UsingLive routing.
type LiveOption func(*liveOpts)

type liveOpts struct {
	failIfNone bool
	require    []Capability
}

// FailIfNone makes ProfileLive/UsingLive error when no provider passes filters.
func FailIfNone() LiveOption {
	return func(o *liveOpts) { o.failIfNone = true }
}

// RequireCaps keeps only providers that advertise all given capabilities.
func RequireCaps(caps ...Capability) LiveOption {
	return func(o *liveOpts) {
		o.require = append([]Capability(nil), caps...)
	}
}

func applyLiveOpts(opts []LiveOption) liveOpts {
	var o liveOpts
	for _, fn := range opts {
		if fn != nil {
			fn(&o)
		}
	}
	return o
}

// FilterHealthy returns names that pass CheckHealth, preserving order.
func (m *Manager) FilterHealthy(ctx context.Context, names ...string) []string {
	if m == nil || len(names) == 0 {
		return nil
	}
	statuses := m.CheckHealth(ctx, names...)
	ok := make(map[string]bool, len(statuses))
	for _, s := range statuses {
		if s.OK && s.Err == nil {
			ok[s.Provider] = true
		}
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		if ok[n] {
			out = append(out, n)
		}
	}
	return out
}

// FilterCapable returns providers that support all required capabilities (order preserved).
func (m *Manager) FilterCapable(names []string, caps ...Capability) []string {
	if m == nil || len(names) == 0 {
		return nil
	}
	if len(caps) == 0 {
		return append([]string(nil), names...)
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" {
			continue
		}
		ok := true
		for _, cap := range caps {
			if !m.Supports(cap, n) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, n)
		}
	}
	return out
}

// ProfileLive returns a profile-scoped client whose fallback chain is filtered
// by health (and optional RequireCaps). If no provider remains and FailIfNone
// is not set, the original profile chain is used.
func (m *Manager) ProfileLive(ctx context.Context, name string, opts ...LiveOption) (*Client, error) {
	if m == nil {
		return nil, fmt.Errorf("ai: manager is nil")
	}
	name = strings.ToLower(strings.TrimSpace(name))
	m.mu.RLock()
	p, ok := m.profileLocked(name)
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("ai: profile [%s] not configured", name)
	}
	if len(p.Providers) == 0 {
		return nil, fmt.Errorf("ai: profile [%s] has no providers", name)
	}
	return m.routeLive(ctx, name, "", p.Providers, opts...)
}

// UsingLive returns a client scoped to the healthy subset of names (order preserved).
func (m *Manager) UsingLive(ctx context.Context, names []string, opts ...LiveOption) (*Client, error) {
	if m == nil {
		return nil, fmt.Errorf("ai: manager is nil")
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("ai: no providers")
	}
	return m.routeLive(ctx, "", "", names, opts...)
}

func (m *Manager) routeLive(ctx context.Context, profile, provider string, names []string, opts ...LiveOption) (*Client, error) {
	o := applyLiveOpts(opts)
	chain := normalizeProviderNames(names)
	filtered := chain
	if len(o.require) > 0 {
		filtered = m.FilterCapable(filtered, o.require...)
	}
	live := m.FilterHealthy(ctx, filtered...)
	if len(live) == 0 {
		if o.failIfNone {
			return nil, fmt.Errorf("ai: no healthy providers")
		}
		live = filtered
		if len(live) == 0 {
			live = chain
		}
	}
	return &Client{
		mgr:       m,
		profile:   profile,
		provider:  provider,
		providers: live,
	}, nil
}
