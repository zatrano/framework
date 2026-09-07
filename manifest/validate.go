package manifest

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// Parse decodes a v1 document. Unknown JSON fields are ignored (forward compatible).
func Parse(raw []byte) (Document, error) {
	var d Document
	if err := json.Unmarshal(raw, &d); err != nil {
		return Document{}, fmt.Errorf("manifest: %w", err)
	}
	if err := Validate(d); err != nil {
		return Document{}, err
	}
	return d, nil
}

// Validate checks a distribution document. It does not consult the addon
// registry or boot the kernel.
func Validate(d Document) error {
	if d.Schema != SchemaV1 {
		return fmt.Errorf("manifest: unsupported schema %q", d.Schema)
	}
	if !validName(d.Name) {
		return fmt.Errorf("manifest: invalid name %q", d.Name)
	}
	if strings.TrimSpace(d.Import) == "" || strings.ContainsAny(d.Import, " \t") {
		return fmt.Errorf("manifest: invalid import %q", d.Import)
	}
	if d.Module != "" && strings.ContainsAny(d.Module, " \t") {
		return fmt.Errorf("manifest: invalid module %q", d.Module)
	}
	switch d.Kind {
	case KindService, KindLibrary:
	default:
		return fmt.Errorf("manifest: invalid kind %q", d.Kind)
	}
	switch d.Layer {
	case LayerFoundation, LayerIntelligence, LayerAddon:
	default:
		return fmt.Errorf("manifest: invalid layer %q", d.Layer)
	}
	if strings.TrimSpace(d.Description) == "" {
		return fmt.Errorf("manifest: missing description")
	}
	switch d.Provider {
	case "", ProviderFactory, ProviderCLI, ProviderFactoryCLI:
	default:
		return fmt.Errorf("manifest: invalid provider %q", d.Provider)
	}
	if d.Kind == KindLibrary && d.Provider == ProviderFactoryCLI {
		// allowed: none today, but factory+cli on a library is valid (future)
	}
	if d.Kind == KindLibrary {
		for _, cap := range d.Capabilities {
			if cap == CapEnable {
				return fmt.Errorf("manifest: library %q cannot have capability %q", d.Name, CapEnable)
			}
		}
	}
	for _, n := range d.Requires {
		if !validName(n) {
			return fmt.Errorf("manifest: invalid requires %q", n)
		}
	}
	for _, n := range d.Optional {
		if !validName(n) {
			return fmt.Errorf("manifest: invalid optional %q", n)
		}
	}
	if min := strings.TrimSpace(d.FrameworkMin); min != "" && !looksLikeSemver(min) {
		return fmt.Errorf("manifest: invalid framework_min %q", d.FrameworkMin)
	}
	known := map[string]bool{CapImport: true, CapEnable: true, CapCLI: true, CapHeavy: true}
	for _, cap := range d.Capabilities {
		if !known[cap] {
			return fmt.Errorf("manifest: unknown capability %q", cap)
		}
	}
	if len(d.Capabilities) > 0 {
		want := deriveCapabilities(d)
		if !capsEqual(d.Capabilities, want) {
			return fmt.Errorf("manifest: capabilities must match kind/provider/heavy")
		}
	}
	if d.Heavy && !strings.Contains(d.Import, d.Name) {
		return fmt.Errorf("manifest: heavy import %q must include name %q", d.Import, d.Name)
	}
	return nil
}

func validName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for i, r := range name {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func looksLikeSemver(v string) bool {
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	if v == "" {
		return false
	}
	parts := strings.Split(strings.Split(strings.Split(v, "-")[0], "+")[0], ".")
	if len(parts) < 1 || len(parts) > 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, r := range p {
			if !unicode.IsDigit(r) {
				return false
			}
		}
	}
	return true
}

func capsEqual(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := map[string]int{}
	for _, c := range got {
		seen[c]++
	}
	for _, c := range want {
		seen[c]--
		if seen[c] < 0 {
			return false
		}
	}
	return true
}
