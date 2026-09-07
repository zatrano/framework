package manifest

import (
	"strings"
)

// SchemaV1 is the only schema this module validates today.
// Future schemas keep the same JSON object with a new "schema" value;
// unknown versions are rejected, unknown fields inside v1 are ignored.
const SchemaV1 = "zatrano.package/v1"

const (
	KindService = "service"
	KindLibrary = "library"

	LayerFoundation   = "foundation"
	LayerIntelligence = "intelligence"
	LayerAddon        = "addon"

	ProviderFactory    = "factory"
	ProviderCLI        = "cli"
	ProviderFactoryCLI = "factory+cli"

	CapImport = "import"
	CapEnable = "enable"
	CapCLI    = "cli"
	CapHeavy  = "heavy"

	DefaultModule    = "github.com/zatrano/packages"
	FrameworkModule  = "github.com/zatrano/framework/v2"
	FrameworkConsole = "github.com/zatrano/framework/v2/console"
)

// Document is the machine-readable package identity for distribution.
//
// Runtime contracts (Register/Boot order, LifecycleProvider, Enabled ∩ Imported)
// are not fields. Publisher, license, and marketplace listing are deferred.
type Document struct {
	Schema       string   `json:"schema"`
	Name         string   `json:"name"`
	Import       string   `json:"import"`
	Module       string   `json:"module,omitempty"`
	Kind         string   `json:"kind"`
	Layer        string   `json:"layer"`
	Heavy        bool     `json:"heavy,omitempty"`
	Description  string   `json:"description"`
	Key          string   `json:"key,omitempty"`
	Requires     []string `json:"requires,omitempty"`
	Optional     []string `json:"optional,omitempty"`
	Provider     string   `json:"provider,omitempty"`
	FrameworkMin string   `json:"framework_min,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// Input is catalog-derived identity used to prototype a Document
// without writing a JSON file per package.
type Input struct {
	Name         string
	Kind         string
	Layer        string
	Description  string
	Heavy        bool
	Key          string
	Requires     []string
	Optional     []string
	Factory      bool
	CLI          bool
	FrameworkMin string
}

// Derive fills import/module/provider/capabilities from catalog facts.
// It does not invent Requires, From(app), or FrameworkMin.
func Derive(in Input) Document {
	name := strings.ToLower(strings.TrimSpace(in.Name))
	kind := strings.ToLower(strings.TrimSpace(in.Kind))
	layer := strings.ToLower(strings.TrimSpace(in.Layer))
	d := Document{
		Schema:       SchemaV1,
		Name:         name,
		Kind:         kind,
		Layer:        layer,
		Heavy:        in.Heavy,
		Description:  strings.TrimSpace(in.Description),
		Key:          strings.ToLower(strings.TrimSpace(in.Key)),
		Requires:     normalizeNames(in.Requires),
		Optional:     normalizeNames(in.Optional),
		FrameworkMin: strings.TrimSpace(in.FrameworkMin),
	}
	switch name {
	case "console":
		d.Import = FrameworkConsole
		d.Module = FrameworkModule
	default:
		d.Module = DefaultModule
		d.Import = DefaultModule + "/" + name
		if in.Heavy {
			d.Module = DefaultModule + "/" + name
			d.Import = d.Module
		}
	}
	switch {
	case in.Factory && in.CLI:
		d.Provider = ProviderFactoryCLI
	case in.Factory:
		d.Provider = ProviderFactory
	case in.CLI:
		d.Provider = ProviderCLI
	}
	d.Capabilities = deriveCapabilities(d)
	return d
}

func deriveCapabilities(d Document) []string {
	caps := []string{CapImport}
	if d.Kind == KindService {
		caps = append(caps, CapEnable)
	}
	if d.Provider == ProviderCLI || d.Provider == ProviderFactoryCLI {
		caps = append(caps, CapCLI)
	}
	if d.Heavy {
		caps = append(caps, CapHeavy)
	}
	return caps
}

func normalizeNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	out := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}
