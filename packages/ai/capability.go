package ai

import (
	"fmt"
	"sort"
	"strings"
)

// Capability is a feature a provider/driver may support.
type Capability string

const (
	CapChat   Capability = "chat"
	CapEmbed  Capability = "embed"
	CapStream Capability = "stream"
	CapTools  Capability = "tools"
	CapJSON   Capability = "json"   // structured output / response_format
	CapVision Capability = "vision" // multimodal image_url message parts
	CapImage  Capability = "image"  // image generation
)

// Capabler lets a driver declare capabilities explicitly.
// When absent, InferCapabilities uses interface probes + known driver types.
type Capabler interface {
	Capabilities() []Capability
}

// ModelInfo is optional metadata for a model id (context window, caps).
type ModelInfo struct {
	ID            string
	ContextWindow int // 0 = unknown
	Caps          []Capability
}

// ProviderInfo describes a registered provider for discovery/admin UI.
type ProviderInfo struct {
	Name         string
	Driver       string // Driver.Name()
	Caps         []Capability
	Models       []ModelInfo
	DefaultModel string
}

// InferCapabilities discovers capabilities for a driver.
func InferCapabilities(d Driver) []Capability {
	if d == nil {
		return nil
	}
	if c, ok := d.(Capabler); ok {
		return normalizeCaps(c.Capabilities())
	}
	caps := []Capability{CapChat}
	if _, ok := d.(EmbeddingDriver); ok {
		caps = append(caps, CapEmbed)
	}
	if _, ok := d.(StreamDriver); ok {
		caps = append(caps, CapStream)
	}
	if _, ok := d.(ImageDriver); ok {
		caps = append(caps, CapImage)
	}
	switch d := d.(type) {
	case FakeDriver, *OpenAIDriver:
		caps = append(caps, CapTools, CapJSON, CapVision, CapImage)
	case *AnthropicDriver:
		caps = append(caps, CapVision)
	case LogDriver:
		inner := d.Inner
		if inner == nil {
			inner = FakeDriver{}
		}
		return InferCapabilities(inner)
	}
	return normalizeCaps(caps)
}

// HasCapability reports whether the driver advertises cap.
func HasCapability(d Driver, cap Capability) bool {
	for _, c := range InferCapabilities(d) {
		if c == cap {
			return true
		}
	}
	return false
}

func normalizeCaps(in []Capability) []Capability {
	seen := make(map[Capability]struct{}, len(in))
	out := make([]Capability, 0, len(in))
	for _, c := range in {
		c = Capability(strings.ToLower(strings.TrimSpace(string(c))))
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// SetModels registers optional model metadata for a named provider.
func (m *Manager) SetModels(provider string, models ...ModelInfo) {
	if m == nil {
		return
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return
	}
	cp := make([]ModelInfo, 0, len(models))
	for _, mod := range models {
		mod.ID = strings.TrimSpace(mod.ID)
		if mod.ID == "" {
			continue
		}
		mod.Caps = normalizeCaps(mod.Caps)
		cp = append(cp, mod)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.models == nil {
		m.models = make(map[string][]ModelInfo)
	}
	m.models[provider] = cp
}

// Capabilities returns sorted capabilities for a provider (default if name omitted).
func (m *Manager) Capabilities(provider ...string) ([]Capability, error) {
	d, err := m.Driver(provider...)
	if err != nil {
		return nil, err
	}
	return InferCapabilities(d), nil
}

// Supports reports whether a provider has the capability.
func (m *Manager) Supports(cap Capability, provider ...string) bool {
	caps, err := m.Capabilities(provider...)
	if err != nil {
		return false
	}
	for _, c := range caps {
		if c == cap {
			return true
		}
	}
	return false
}

// Describe returns discovery metadata for a provider (default if name omitted).
func (m *Manager) Describe(provider ...string) (ProviderInfo, error) {
	if m == nil {
		return ProviderInfo{}, fmt.Errorf("ai: manager is nil")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := m.defaultDriver
	if len(provider) > 0 && strings.TrimSpace(provider[0]) != "" {
		n = strings.ToLower(strings.TrimSpace(provider[0]))
	}
	d, ok := m.drivers[n]
	if !ok {
		return ProviderInfo{}, fmt.Errorf("ai: driver [%s] not configured", n)
	}
	info := ProviderInfo{
		Name:   n,
		Driver: d.Name(),
		Caps:   InferCapabilities(d),
	}
	if mods, ok := m.models[n]; ok {
		info.Models = append([]ModelInfo(nil), mods...)
	}
	info.DefaultModel = m.defaults.Model
	switch od := d.(type) {
	case *OpenAIDriver:
		if od.Model != "" {
			info.DefaultModel = od.Model
		}
	case *AnthropicDriver:
		if od.Model != "" {
			info.DefaultModel = od.Model
		}
	case *GeminiDriver:
		if od.Model != "" {
			info.DefaultModel = od.Model
		}
	}
	return info, nil
}

// DescribeAll returns ProviderInfo for every registered provider (sorted by name).
func (m *Manager) DescribeAll() []ProviderInfo {
	if m == nil {
		return nil
	}
	names := m.Drivers()
	out := make([]ProviderInfo, 0, len(names))
	for _, name := range names {
		info, err := m.Describe(name)
		if err != nil {
			continue
		}
		out = append(out, info)
	}
	return out
}
