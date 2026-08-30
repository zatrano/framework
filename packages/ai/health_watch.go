package ai

import (
	"context"
	"time"
)

// HealthWatchConfig controls background health probes that rewrite profile provider order.
type HealthWatchConfig struct {
	Interval    time.Duration // default 30s
	Profiles    []string      // empty = all registered profiles
	OnlyHealthy bool          // drop unhealthy providers; otherwise healthy-first reorder
	FailOpen    bool          // if no healthy left, keep previous chain (default true when OnlyHealthy)
	OnUpdate    func(profile string, providers []string)
}

// WatchHealth runs until ctx is canceled. First probe runs immediately.
// It mutates profile provider lists via SetProfile (original order among healthy preserved).
func (m *Manager) WatchHealth(ctx context.Context, cfg HealthWatchConfig) error {
	if m == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	failOpen := cfg.FailOpen
	if !cfg.OnlyHealthy {
		failOpen = true
	}

	run := func() {
		m.refreshProfilesFromHealth(ctx, cfg.Profiles, cfg.OnlyHealthy, failOpen, cfg.OnUpdate)
	}
	run()

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			run()
		}
	}
}

func (m *Manager) refreshProfilesFromHealth(ctx context.Context, profiles []string, onlyHealthy, failOpen bool, onUpdate func(string, []string)) {
	if len(profiles) == 0 {
		profiles = m.Profiles()
	}
	if len(profiles) == 0 {
		return
	}
	// Collect union of providers to probe once.
	need := map[string]struct{}{}
	type prof struct {
		name string
		p    Profile
	}
	list := make([]prof, 0, len(profiles))
	m.mu.RLock()
	for _, name := range profiles {
		p, ok := m.profileLocked(name)
		if !ok || len(p.Providers) == 0 {
			continue
		}
		list = append(list, prof{name: name, p: p})
		for _, pr := range p.Providers {
			need[pr] = struct{}{}
		}
	}
	m.mu.RUnlock()
	if len(list) == 0 {
		return
	}
	names := make([]string, 0, len(need))
	for n := range need {
		names = append(names, n)
	}
	statuses := m.CheckHealth(ctx, names...)
	ok := map[string]bool{}
	for _, s := range statuses {
		ok[s.Provider] = s.OK && s.Err == nil
	}

	for _, item := range list {
		healthy, unhealthy := splitByHealth(item.p.Providers, ok)
		var next []string
		if onlyHealthy {
			next = healthy
			if len(next) == 0 {
				if failOpen {
					continue
				}
				next = item.p.Providers
			}
		} else {
			next = append(append([]string{}, healthy...), unhealthy...)
		}
		if sameStringSlice(next, item.p.Providers) {
			continue
		}
		updated := item.p
		updated.Providers = next
		m.SetProfile(item.name, updated)
		if onUpdate != nil {
			onUpdate(item.name, next)
		}
	}
}

func splitByHealth(providers []string, ok map[string]bool) (healthy, unhealthy []string) {
	for _, p := range providers {
		if ok[p] {
			healthy = append(healthy, p)
		} else {
			unhealthy = append(unhealthy, p)
		}
	}
	return healthy, unhealthy
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
