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
	// Requires lists addon names that must be imported and must Register/Boot
	// first. A missing requirement is a startup error, not a skip.
	Requires []string
	// Optional lists addon names that boot first when they are imported, and
	// are ignored when they are not.
	Optional []string
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
	m.Requires = normalizeDeps(m.Requires)
	m.Optional = normalizeDeps(m.Optional)

	registryMu.Lock()
	defer registryMu.Unlock()
	for _, existing := range registry {
		if existing.Name == name {
			panic("addons: duplicate registration: " + name)
		}
	}
	registry = append(registry, m)
}

// ClearRegistry removes every registered addon. Tests use this to isolate
// enablement fixtures; production code must not call it.
func ClearRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = nil
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

// Resolve expands Requires/Optional from the process registry, then returns
// metas in boot order. Unknown names and missing Requires are errors.
func Resolve(names ...string) ([]Meta, error) {
	selected, err := Expand(names, Lookup)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, nil
	}
	return OrderMetas(selected)
}

// Select builds providers for the given addon names (unknown names error).
// Required dependencies that are imported are pulled in automatically.
func Select(names ...string) ([]contracts.Provider, error) {
	ordered, err := Resolve(names...)
	if err != nil {
		return nil, err
	}
	if len(ordered) == 0 {
		return nil, nil
	}
	return providersOf(ordered), nil
}

// Expand closes a name set over Requires (must exist) and Optional (if present).
func Expand(names []string, lookup func(string) (Meta, bool)) ([]Meta, error) {
	if lookup == nil {
		lookup = Lookup
	}
	if len(names) == 0 {
		return nil, nil
	}
	seen := map[string]Meta{}
	queue := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		queue = append(queue, name)
	}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if _, ok := seen[name]; ok {
			continue
		}
		m, ok := lookup(name)
		if !ok {
			return nil, fmt.Errorf("unknown addon package %q (run package:list)", name)
		}
		seen[name] = m
		for _, req := range m.Requires {
			if req == "" || req == name {
				continue
			}
			if _, already := seen[req]; already {
				continue
			}
			if _, ok := lookup(req); !ok {
				return nil, fmt.Errorf("addons: %q requires %q, which is not imported", name, req)
			}
			queue = append(queue, req)
		}
		for _, opt := range m.Optional {
			if opt == "" || opt == name {
				continue
			}
			if _, already := seen[opt]; already {
				continue
			}
			if _, ok := lookup(opt); ok {
				queue = append(queue, opt)
			}
		}
	}
	out := make([]Meta, 0, len(seen))
	for _, m := range seen {
		out = append(out, m)
	}
	return out, nil
}

// Bootable drops addons whose Meta.Requires are not in the same set, then
// repeats until the graph is stable. A Go import of a helper (for example
// validation pulling flash) must not crash App() because session was not
// imported. Explicit Select/WithAddons still error on missing Requires.
func Bootable(metas []Meta) []Meta {
	byName := make(map[string]Meta, len(metas))
	for _, m := range metas {
		if m.Name == "" {
			continue
		}
		byName[m.Name] = m
	}
	for {
		next := make(map[string]Meta, len(byName))
		dropped := false
		for name, m := range byName {
			ok := true
			for _, req := range m.Requires {
				if req == "" || req == name {
					continue
				}
				if _, has := byName[req]; !has {
					ok = false
					break
				}
			}
			if ok {
				next[name] = m
			} else {
				dropped = true
			}
		}
		byName = next
		if !dropped {
			break
		}
	}
	out := make([]Meta, 0, len(byName))
	for _, m := range byName {
		out = append(out, m)
	}
	return out
}

// DefaultMetas is the default enable-set: every imported addon whose
// Requires are present, in boot order.
func DefaultMetas() []Meta {
	ordered, err := OrderMetas(Bootable(Available()))
	if err != nil {
		panic(err)
	}
	return ordered
}

// DefaultPackageProviders returns DefaultMetas factories in boot order.
func DefaultPackageProviders() []contracts.Provider {
	return providersOf(DefaultMetas())
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

// OrderMetas returns metas in boot order. Requires missing from the input set
// are errors. Optional dependencies participate only when present. Ties use
// Order then Name.
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
		seenDep := map[string]bool{}
		for _, req := range m.Requires {
			if req == "" || req == m.Name || seenDep[req] {
				continue
			}
			seenDep[req] = true
			if _, ok := byName[req]; !ok {
				return nil, fmt.Errorf("addons: %q requires %q, which is not in the boot set", m.Name, req)
			}
			graph[req] = append(graph[req], m.Name)
			incoming[m.Name]++
		}
		for _, opt := range m.Optional {
			if opt == "" || opt == m.Name || seenDep[opt] {
				continue
			}
			seenDep[opt] = true
			if _, ok := byName[opt]; !ok {
				continue
			}
			graph[opt] = append(graph[opt], m.Name)
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

func normalizeDeps(deps []string) []string {
	if len(deps) == 0 {
		return nil
	}
	out := make([]string, 0, len(deps))
	seen := map[string]bool{}
	for _, dep := range deps {
		dep = strings.ToLower(strings.TrimSpace(dep))
		if dep == "" || seen[dep] {
			continue
		}
		seen[dep] = true
		out = append(out, dep)
	}
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
