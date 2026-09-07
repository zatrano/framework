package acquire

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zatrano/framework/v2/registry"
)

const SchemaV1 = "zatrano.acquire/v1"

// Plan is how a resolved identity becomes a go get argument.
// It is not dependency state, not enablement, and not a lockfile.
// Name/Import are catalog identity copied from Result; they do not mean
// the package is enabled. After a later Apply, go.mod / go.sum are authoritative.
type Plan struct {
	Schema   string `json:"schema"`
	Name     string `json:"name"`
	Import   string `json:"import"`
	Module   string `json:"module"`
	Query    string `json:"query"`
	Selected string `json:"selected"`
	Kind     string `json:"kind,omitempty"`
	Heavy    bool   `json:"heavy,omitempty"`
}

// FromResult maps a successful registry.Result onto a module acquisition plan.
// It does not call Resolve, re-check framework_min, run go get, or touch the filesystem.
func FromResult(got registry.Result) (Plan, error) {
	name := strings.ToLower(strings.TrimSpace(got.Package.Name))
	mod := strings.TrimSpace(got.Package.Module)
	if name == "" || mod == "" {
		return Plan{}, fmt.Errorf("acquire: result missing name or module")
	}
	selected, query, err := moduleQuery(mod, got.Release)
	if err != nil {
		return Plan{}, err
	}
	return Plan{
		Schema:   SchemaV1,
		Name:     name,
		Import:   strings.TrimSpace(got.Package.Import),
		Module:   mod,
		Query:    query,
		Selected: selected,
		Kind:     strings.ToLower(strings.TrimSpace(got.Package.Kind)),
		Heavy:    got.Package.Heavy,
	}, nil
}

// GoGetArg is the token later passed to `go get`. It is never the selector "latest".
func (p Plan) GoGetArg() string {
	return p.Query
}

// Targets is module-level normalization: unique Query per Module, ordered by
// module path. It deduplicates; it does not resolve conflicts or run go get.
// session@main and auth@main collapse to one query. session@main and auth@v1.0.0
// on the same module are an error — not last-write-wins, not a second Resolve.
func Targets(plans []Plan) ([]string, error) {
	byMod := make(map[string]string, len(plans))
	for _, p := range plans {
		mod := strings.TrimSpace(p.Module)
		q := strings.TrimSpace(p.Query)
		if mod == "" || q == "" {
			return nil, fmt.Errorf("acquire: incomplete plan")
		}
		if prev, ok := byMod[mod]; ok && prev != q {
			return nil, fmt.Errorf("acquire: conflicting plans for %s: %s vs %s", mod, prev, q)
		}
		byMod[mod] = q
	}
	mods := make([]string, 0, len(byMod))
	for mod := range byMod {
		mods = append(mods, mod)
	}
	sort.Strings(mods)
	out := make([]string, 0, len(mods))
	for _, mod := range mods {
		out = append(out, byMod[mod])
	}
	return out, nil
}

func moduleQuery(mod string, rel registry.Release) (selected, query string, err error) {
	if v := strings.TrimSpace(rel.Version); v != "" {
		v = strings.TrimPrefix(strings.ToLower(v), "v")
		if v == "" || strings.EqualFold(v, "latest") {
			return "", "", fmt.Errorf("acquire: invalid tagged version %q", rel.Version)
		}
		return v, mod + "@v" + v, nil
	}
	ch := strings.TrimSpace(rel.Channel)
	if strings.EqualFold(ch, registry.ChannelMain) {
		return registry.ChannelMain, mod + "@" + registry.ChannelMain, nil
	}
	return "", "", fmt.Errorf("acquire: release needs a version or channel main")
}
