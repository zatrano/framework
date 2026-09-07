package registry

import (
	"fmt"
	"strings"

	"github.com/zatrano/framework/v2/manifest"
)

const SchemaV1 = "zatrano.registry/v1"

const ChannelMain = "main"

// Package is a stable identity in the index. Versions live on the Go module,
// not as a second number beside the catalog name.
type Package struct {
	Name        string    `json:"name"`
	Import      string    `json:"import"`
	Module      string    `json:"module"`
	Kind        string    `json:"kind"`
	Layer       string    `json:"layer"`
	Heavy       bool      `json:"heavy,omitempty"`
	Description string    `json:"description"`
	Releases    []Release `json:"releases"`
}

// Release is one discoverable artifact of Package.Module.
type Release struct {
	Version      string `json:"version,omitempty"`
	Channel      string `json:"channel,omitempty"`
	FrameworkMin string `json:"framework_min,omitempty"`
	Digest       string `json:"digest,omitempty"`
}

// Index is the authoritative in-memory catalog. A later HTTP service should
// serialize this document, not a marketplace listing.
type Index struct {
	Schema   string    `json:"schema"`
	Packages []Package `json:"packages"`
}

// Query selects one release. Install/update behavior is not defined here.
type Query struct {
	Name      string
	Version   string // empty/"latest", "main", or tagged semver
	Framework string
	Kind      string
}

// Result is a resolved identity plus the chosen release.
type Result struct {
	Package Package
	Release Release
}

// Filter is discovery without version picking.
type Filter struct {
	Query string
	Kind  string
	Layer string
	Heavy *bool
}

// FromDocuments builds an index from validated manifests. Official untagged
// modules get channel "main". Duplicate names or imports fail.
func FromDocuments(docs []manifest.Document) (Index, error) {
	idx := Index{Schema: SchemaV1, Packages: make([]Package, 0, len(docs))}
	names := map[string]bool{}
	imports := map[string]bool{}
	for _, d := range docs {
		if err := manifest.Validate(d); err != nil {
			return Index{}, err
		}
		if names[d.Name] {
			return Index{}, fmt.Errorf("registry: duplicate name %q", d.Name)
		}
		if imports[d.Import] {
			return Index{}, fmt.Errorf("registry: duplicate import %q", d.Import)
		}
		names[d.Name] = true
		imports[d.Import] = true
		mod := d.Module
		if mod == "" {
			mod = manifest.DefaultModule
		}
		idx.Packages = append(idx.Packages, Package{
			Name:        d.Name,
			Import:      d.Import,
			Module:      mod,
			Kind:        d.Kind,
			Layer:       d.Layer,
			Heavy:       d.Heavy,
			Description: d.Description,
			Releases: []Release{{
				Channel:      ChannelMain,
				FrameworkMin: d.FrameworkMin,
			}},
		})
	}
	if err := idx.Validate(); err != nil {
		return Index{}, err
	}
	return idx, nil
}

// Validate checks index uniqueness and release shape. It does not boot packages.
func (idx Index) Validate() error {
	if idx.Schema != SchemaV1 {
		return fmt.Errorf("registry: unsupported schema %q", idx.Schema)
	}
	names := map[string]bool{}
	imports := map[string]bool{}
	for _, p := range idx.Packages {
		if p.Name == "" || p.Import == "" || p.Module == "" {
			return fmt.Errorf("registry: package missing name/import/module")
		}
		if names[p.Name] {
			return fmt.Errorf("registry: duplicate name %q", p.Name)
		}
		if imports[p.Import] {
			return fmt.Errorf("registry: duplicate import %q", p.Import)
		}
		names[p.Name] = true
		imports[p.Import] = true
		if p.Kind != manifest.KindService && p.Kind != manifest.KindLibrary {
			return fmt.Errorf("registry: %q invalid kind %q", p.Name, p.Kind)
		}
		if len(p.Releases) == 0 {
			return fmt.Errorf("registry: %q has no releases", p.Name)
		}
		for _, r := range p.Releases {
			if r.Version == "" && r.Channel == "" {
				return fmt.Errorf("registry: %q release needs version or channel", p.Name)
			}
		}
	}
	return nil
}

func (idx Index) Lookup(name string) (Package, bool) {
	want := strings.ToLower(strings.TrimSpace(name))
	for _, p := range idx.Packages {
		if p.Name == want {
			return p, true
		}
	}
	return Package{}, false
}
