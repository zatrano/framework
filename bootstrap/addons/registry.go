package addons

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/zatrano/framework/contracts"
)

// Meta describes a first-party addon package for discovery and CLI.
type Meta struct {
	Name        string
	Key         string // container binding key
	Description string
	Heavy       bool
	Factory     func() contracts.Provider
	// Order is a tie-breaker among independent addons (lower first).
	// Real boot order is computed from Requires via topological sort.
	Order int
	// Requires lists addon names that must Register/Boot first when they are
	// also present in the process registry. Missing names are skipped (opt-in).
	Requires []string
	// CLI, if set, is invoked from console.New when this addon is imported.
	CLI func(app contracts.App) []CLICommand
}

// CLICommand is an addon-provided console command.
type CLICommand struct {
	Name        string
	Description string
	Handle      func(args []string) error
}

// registry is process-global: blank-imports register into this process, not
// into a specific Application. Two Application values in one process share it.
// WithAddons selects a subset for one app; it does not isolate the registry.
var (
	registryMu sync.RWMutex
	registry   []Meta
)

// Register adds an addon factory. Duplicate names panic so init order cannot
// silently discard a second package that reused a name.
// Addon packages call this from init(); this package must not import those addons.
func Register(m Meta) {
	name := strings.ToLower(strings.TrimSpace(m.Name))
	if name == "" {
		return
	}
	if m.Key == "" {
		m.Key = name
	}
	m.Name = name
	for i, req := range m.Requires {
		m.Requires[i] = strings.ToLower(strings.TrimSpace(req))
	}

	registryMu.Lock()
	defer registryMu.Unlock()
	for _, existing := range registry {
		if existing.Name == name {
			panic("addons: duplicate registration: " + name)
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
func Select(names ...string) ([]contracts.Provider, error) {
	if len(names) == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	var selected []Meta
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
		selected = append(selected, m)
	}
	ordered, err := OrderMetas(selected)
	if err != nil {
		return nil, err
	}
	return providersOf(ordered), nil
}

// DefaultPackageProviders returns every registered addon provider in dependency order.
func DefaultPackageProviders() []contracts.Provider {
	avail := Available()
	ordered, err := OrderMetas(avail)
	if err != nil {
		panic(err)
	}
	return providersOf(ordered)
}

func providersOf(metas []Meta) []contracts.Provider {
	out := make([]contracts.Provider, 0, len(metas))
	for _, m := range metas {
		if m.Factory == nil {
			continue
		}
		out = append(out, m.Factory())
	}
	return out
}

// AllNames is the ordered default enable-set for full demo apps.
func AllNames() []string {
	return Names()
}

// OrderMetas returns metas in boot order: Requires form a graph; missing
// requirements (not in the input set) are ignored. Ties use Order then Name.
func OrderMetas(metas []Meta) ([]Meta, error) {
	byName := make(map[string]Meta, len(metas))
	incoming := make(map[string]int, len(metas))
	graph := make(map[string][]string, len(metas))
	for _, m := range metas {
		if _, ok := byName[m.Name]; ok {
			return nil, fmt.Errorf("addons: duplicate name %q in order set", m.Name)
		}
		byName[m.Name] = m
		incoming[m.Name] = 0
	}
	for _, m := range metas {
		seenReq := map[string]bool{}
		for _, req := range m.Requires {
			if req == "" || req == m.Name || seenReq[req] {
				continue
			}
			seenReq[req] = true
			if _, ok := byName[req]; !ok {
				continue
			}
			graph[req] = append(graph[req], m.Name)
			incoming[m.Name]++
		}
	}
	ready := make([]Meta, 0, len(metas))
	for _, m := range metas {
		if incoming[m.Name] == 0 {
			ready = append(ready, m)
		}
	}
	sortReady := func(items []Meta) {
		sort.Slice(items, func(i, j int) bool {
			if items[i].Order != items[j].Order {
				return items[i].Order < items[j].Order
			}
			return items[i].Name < items[j].Name
		})
	}
	out := make([]Meta, 0, len(metas))
	for len(ready) > 0 {
		sortReady(ready)
		next := ready[0]
		ready = ready[1:]
		out = append(out, next)
		for _, dep := range graph[next.Name] {
			incoming[dep]--
			if incoming[dep] == 0 {
				ready = append(ready, byName[dep])
			}
		}
	}
	if len(out) != len(metas) {
		return nil, fmt.Errorf("addons: dependency cycle among %v", namesOf(metas))
	}
	return out, nil
}

func namesOf(metas []Meta) []string {
	out := make([]string, len(metas))
	for i, m := range metas {
		out[i] = m.Name
	}
	sort.Strings(out)
	return out
}

// Plan is the package lifecycle snapshot for one process.
//
//	imported  = blank-import init() registry (process-global)
//	enabled   = WithAddons subset, or every imported name when WithAddons is omitted
//	booted    = after Application.Bootstrap (providers Register+Boot)
type Plan struct {
	Imported []string
	Enabled  []string
}

// NewPlan builds a lifecycle snapshot. A nil enabled slice means "all imported".
// An empty non-nil slice means kernel-only (WithAddons()).
func NewPlan(enabled []string) Plan {
	imported := Names()
	p := Plan{Imported: imported}
	if enabled == nil {
		p.Enabled = append([]string(nil), imported...)
		return p
	}
	p.Enabled = make([]string, 0, len(enabled))
	seen := map[string]bool{}
	for _, name := range enabled {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		p.Enabled = append(p.Enabled, name)
	}
	sort.Strings(p.Enabled)
	return p
}
