package addons

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zatrano/framework/core"
)

// Meta describes a first-party addon package for discovery and CLI.
type Meta struct {
	Name        string
	Key         string // container binding key
	Description string
	Heavy       bool
	Factory     func() core.Provider
}

var registry = []Meta{
	{Name: "features", Key: "features", Description: "Feature flags and gradual rollouts", Factory: func() core.Provider { return &FeaturesServiceProvider{} }},
	{Name: "tenancy", Key: "tenancy", Description: "Multi-tenant resolution", Factory: func() core.Provider { return &TenancyServiceProvider{} }},
	{Name: "graphql", Key: "graphql", Description: "GraphQL schema and queries", Factory: func() core.Provider { return &GraphQLServiceProvider{} }},
	{Name: "audit", Key: "audit", Description: "Request/audit event log", Factory: func() core.Provider { return &AuditServiceProvider{} }},
	{Name: "webhooks", Key: "webhooks", Description: "Outbound signed webhooks", Factory: func() core.Provider { return &WebhooksServiceProvider{} }},
	{Name: "inspector", Key: "inspector", Description: "Request inspector toolbar data", Factory: func() core.Provider { return &InspectorServiceProvider{} }},
	{Name: "search", Key: "search", Description: "In-memory search engine", Factory: func() core.Provider { return &SearchServiceProvider{} }},
	{Name: "social", Key: "social", Description: "Social OAuth login (GitHub/Google)", Factory: func() core.Provider { return &SocialServiceProvider{} }},
	{Name: "enums", Key: "enums", Description: "String enum registry", Factory: func() core.Provider { return &EnumsServiceProvider{} }},
	{Name: "bus", Key: "bus", Description: "Synchronous command bus", Factory: func() core.Provider { return &BusServiceProvider{} }},
	{Name: "pulse", Key: "pulse", Description: "Metrics pulse dashboard", Factory: func() core.Provider { return &PulseServiceProvider{} }},
	{Name: "backup", Key: "backup", Description: "Database backup/restore (SQLite + native dump tools)", Factory: func() core.Provider { return &BackupServiceProvider{} }},
	{Name: "docs", Key: "docs", Description: "Markdown docs repository", Factory: func() core.Provider { return &DocsServiceProvider{} }},
	{Name: "billing", Key: "billing", Description: "Central billing (memory/stripe gateways, webhooks)", Factory: func() core.Provider { return &BillingServiceProvider{} }},
	{Name: "mongo", Key: "mongo", Description: "MongoDB client (separate module)", Heavy: true, Factory: func() core.Provider { return &MongoServiceProvider{} }},
	{Name: "oauth", Key: "oauth", Description: "OAuth2 authorization server", Factory: func() core.Provider { return &OAuthServiceProvider{} }},
	{Name: "octane", Key: "octane", Description: "Concurrent runtime metrics", Factory: func() core.Provider { return &OctaneServiceProvider{} }},
	{Name: "ai", Key: "ai", Description: "AI chat providers", Factory: func() core.Provider { return &AIServiceProvider{} }},
	{Name: "sitemap", Key: "sitemap", Description: "XML sitemap builder", Factory: func() core.Provider { return &SitemapServiceProvider{} }},
	{Name: "lock", Key: "lock", Description: "Atomic locks", Factory: func() core.Provider { return &LockServiceProvider{} }},
	{Name: "circuit", Key: "circuit", Description: "Circuit breaker", Factory: func() core.Provider { return &CircuitServiceProvider{} }},
	{Name: "hashid", Key: "hashid", Description: "Obfuscated public IDs", Factory: func() core.Provider { return &HashIDServiceProvider{} }},
	{Name: "shorturl", Key: "shorturl", Description: "Short URL manager", Factory: func() core.Provider { return &ShortURLServiceProvider{} }},
	{Name: "wellknown", Key: "wellknown", Description: "security.txt / well-known", Factory: func() core.Provider { return &WellKnownServiceProvider{} }},
	{Name: "geo", Key: "geo", Description: "Geolocation resolver", Factory: func() core.Provider { return &GeoServiceProvider{} }},
	{Name: "webauthn", Key: "webauthn", Description: "WebAuthn/passkeys (separate module)", Heavy: true, Factory: func() core.Provider { return &WebAuthnServiceProvider{} }},
	{Name: "otp", Key: "otp", Description: "One-time passwords", Factory: func() core.Provider { return &OTPServiceProvider{} }},
}

// Available returns addon metadata sorted by name.
func Available() []Meta {
	out := append([]Meta(nil), registry...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Lookup finds addon meta by name (case-insensitive).
func Lookup(name string) (Meta, bool) {
	want := strings.ToLower(strings.TrimSpace(name))
	for _, m := range registry {
		if m.Name == want {
			return m, true
		}
	}
	return Meta{}, false
}

// Names returns all registered addon names.
func Names() []string {
	out := make([]string, 0, len(registry))
	for _, m := range registry {
		out = append(out, m.Name)
	}
	sort.Strings(out)
	return out
}

// Select builds providers for the given addon names (unknown names error).
func Select(names ...string) ([]core.Provider, error) {
	if len(names) == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	out := make([]core.Provider, 0, len(names))
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
func DefaultPackageProviders() []core.Provider {
	out := make([]core.Provider, 0, len(registry))
	for _, m := range Available() {
		out = append(out, m.Factory())
	}
	return out
}

// AllNames is the ordered default enable-set for full demo apps.
func AllNames() []string {
	return Names()
}
