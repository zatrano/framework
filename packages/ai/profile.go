package ai

import "strings"

// Profile is a named AI use-case: provider chain + request overrides.
type Profile struct {
	// Providers is an ordered fallback list of registered driver/provider names.
	Providers   []string
	Model       string
	Temperature *float64
	MaxTokens   int
}

func (p Profile) clone() Profile {
	out := p
	if p.Providers != nil {
		out.Providers = append([]string(nil), p.Providers...)
	}
	if p.Temperature != nil {
		t := *p.Temperature
		out.Temperature = &t
	}
	return out
}

func normalizeProviderNames(names []string) []string {
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
