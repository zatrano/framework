package acquire

import (
	"fmt"
	"strings"

	"github.com/zatrano/framework/v2/registry"
)

const SchemaV1 = "zatrano.acquire/v1"

// Plan is how a resolved identity becomes a go get argument.
// It is not dependency state. After Apply, go.mod / go.sum are authoritative.
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

// FromResult maps registry.Resolve output onto a module acquisition plan.
// It does not call Resolve, run go get, or write go.mod.
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

// GoGetArg is the token passed to `go get`. It is never the selector "latest".
func (p Plan) GoGetArg() string {
	return p.Query
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
