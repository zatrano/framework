package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Manager resolves named AI providers and profiles (SMS-style registry).
type Manager struct {
	mu            sync.RWMutex
	defaultDriver string
	drivers       map[string]Driver
	profiles      map[string]Profile
	defaults      Defaults
}

// New creates an AI manager with fake and log drivers.
func New() *Manager {
	m := &Manager{
		drivers:       make(map[string]Driver),
		profiles:      make(map[string]Profile),
		defaultDriver: "fake",
		defaults:      Defaults{Timeout: 30 * time.Second},
	}
	m.Extend("fake", FakeDriver{})
	m.Extend("log", LogDriver{})
	return m
}

// SetDefaults configures request defaults (model, sampling, timeout).
func (m *Manager) SetDefaults(d Defaults) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d.Timeout <= 0 {
		d.Timeout = 30 * time.Second
	}
	m.defaults = d
}

// Defaults returns a copy of manager defaults.
func (m *Manager) Defaults() Defaults {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaults
}

// Extend registers a named provider/driver.
func (m *Manager) Extend(name string, driver Driver) {
	if m == nil || name == "" || driver == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.drivers[strings.ToLower(strings.TrimSpace(name))] = driver
}

// Use sets the default provider name.
func (m *Manager) Use(name string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultDriver = strings.ToLower(strings.TrimSpace(name))
}

// SetProfile registers or replaces a named profile.
func (m *Manager) SetProfile(name string, profile Profile) {
	if m == nil || strings.TrimSpace(name) == "" {
		return
	}
	profile.Providers = normalizeProviderNames(profile.Providers)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[strings.ToLower(strings.TrimSpace(name))] = profile.clone()
}

// Profiles returns registered profile names.
func (m *Manager) Profiles() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.profiles))
	for name := range m.profiles {
		out = append(out, name)
	}
	return out
}

// Drivers returns registered provider names.
func (m *Manager) Drivers() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.drivers))
	for name := range m.drivers {
		out = append(out, name)
	}
	return out
}

// Driver returns a named provider (or the default).
func (m *Manager) Driver(name ...string) (Driver, error) {
	if m == nil {
		return nil, fmt.Errorf("ai: manager is nil")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := m.defaultDriver
	if len(name) > 0 && strings.TrimSpace(name[0]) != "" {
		n = strings.ToLower(strings.TrimSpace(name[0]))
	}
	d, ok := m.drivers[n]
	if !ok {
		return nil, fmt.Errorf("ai: driver [%s] not configured", n)
	}
	return d, nil
}

// Using returns a client scoped to a named provider.
func (m *Manager) Using(name string) *Client {
	return &Client{mgr: m, provider: strings.ToLower(strings.TrimSpace(name))}
}

// Profile returns a client scoped to a named profile (fallback chain + overrides).
func (m *Manager) Profile(name string) *Client {
	return &Client{mgr: m, profile: strings.ToLower(strings.TrimSpace(name))}
}

// Chat runs a chat completion on the default (or named) provider.
func (m *Manager) Chat(ctx context.Context, req ChatRequest, driver ...string) (*ChatResponse, error) {
	if len(driver) > 0 && strings.TrimSpace(driver[0]) != "" {
		return m.Using(driver[0]).Chat(ctx, req)
	}
	return (&Client{mgr: m}).Chat(ctx, req)
}

// Embed runs embeddings on the default (or named) provider when supported.
func (m *Manager) Embed(ctx context.Context, req EmbedRequest, driver ...string) (*EmbedResponse, error) {
	if len(driver) > 0 && strings.TrimSpace(driver[0]) != "" {
		return m.Using(driver[0]).Embed(ctx, req)
	}
	return (&Client{mgr: m}).Embed(ctx, req)
}

func (m *Manager) profileLocked(name string) (Profile, bool) {
	p, ok := m.profiles[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return Profile{}, false
	}
	return p.clone(), true
}

func applyDefaults(req ChatRequest, defs Defaults) ChatRequest {
	if req.Model == "" {
		req.Model = defs.Model
	}
	if req.Temperature == nil && defs.Temperature != nil {
		t := *defs.Temperature
		req.Temperature = &t
	}
	if req.MaxTokens <= 0 && defs.MaxTokens > 0 {
		req.MaxTokens = defs.MaxTokens
	}
	return req
}

func ensureModel(req ChatRequest) ChatRequest {
	if req.Model == "" {
		req.Model = "zatrano-fake-1"
	}
	return req
}

func applyProfile(req ChatRequest, p Profile) ChatRequest {
	if req.Model == "" && p.Model != "" {
		req.Model = p.Model
	}
	if req.Temperature == nil && p.Temperature != nil {
		t := *p.Temperature
		req.Temperature = &t
	}
	if req.MaxTokens <= 0 && p.MaxTokens > 0 {
		req.MaxTokens = p.MaxTokens
	}
	return req
}
