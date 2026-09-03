package addons

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/zatrano/framework/kernel"
)

// Meta describes a first-party addon package for discovery and CLI.
type Meta struct {
	Name        string
	Key         string // container binding key
	Description string
	Heavy       bool
	Factory     func() kernel.Provider
}

var (
	registryMu sync.RWMutex
	registry   []Meta
)

// Register adds an addon factory. Duplicate names are ignored (first registration wins).
// Addon packages call this from init(); this package must not import those addons.
func Register(m Meta) {
	name := strings.ToLower(strings.TrimSpace(m.Name))
	if name == "" || m.Factory == nil {
		return
	}
	if m.Key == "" {
		m.Key = name
	}
	m.Name = name

	registryMu.Lock()
	defer registryMu.Unlock()
	for _, existing := range registry {
		if existing.Name == name {
			return
		}
	}
	registry = append(registry, m)
}

// Available returns addon metadata sorted by name.
func Available() []Meta {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := append([]Meta(nil), registry...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Lookup finds addon meta by name (case-insensitive).
func Lookup(name string) (Meta, bool) {
	want := strings.ToLower(strings.TrimSpace(name))
	registryMu.RLock()
	defer registryMu.RUnlock()
	for _, m := range registry {
		if m.Name == want {
			return m, true
		}
	}
	return Meta{}, false
}

// Names returns all registered addon names.
func Names() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]string, 0, len(registry))
	for _, m := range registry {
		out = append(out, m.Name)
	}
	sort.Strings(out)
	return out
}

// Select builds providers for the given addon names (unknown names error).
func Select(names ...string) ([]kernel.Provider, error) {
	if len(names) == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	out := make([]kernel.Provider, 0, len(names))
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || seen[name] {
			continue
		}
		m, ok := Lookup(name)
		if !ok {
			return nil, fmt.Errorf("unknown addon package %q (run package:list)", name)
		}
		seen[name] = true
		out = append(out, m.Factory())
	}
	return out, nil
}

// DefaultPackageProviders returns every registered addon provider (demo/full stack).
func DefaultPackageProviders() []kernel.Provider {
	avail := Available()
	out := make([]kernel.Provider, 0, len(avail))
	for _, m := range avail {
		out = append(out, m.Factory())
	}
	return out
}

// AllNames is the ordered default enable-set for full demo apps.
func AllNames() []string {
	return Names()
}
