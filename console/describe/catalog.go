package describe

import (
	"sort"

	"github.com/zatrano/framework/v2/kernel"
)

// ecosystemCatalog is the CLI aggregation of packages-module names.
// The kernel catalog stays primitive-only; this list is how package:list,
// doctor, and describe know foundation / intelligence / addon packages
// without the kernel importing them.
var ecosystemCatalog = []kernel.PackageInfo{
	{Name: "session", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "HTTP sessions"},
	{Name: "flash", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Flash / toast messages"},
	{Name: "validation", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Input validation"},
	{Name: "auth", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Authentication guards"},
	{Name: "authorization", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Gates and policies"},
	{Name: "hashing", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Password hashing"},
	{Name: "cache", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Cache manager"},
	{Name: "redisx", Layer: kernel.LayerFoundation, Kind: kernel.KindLibrary, Description: "Redis client helper (cache owns the connection)"},
	{Name: "database", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Database manager"},
	{Name: "orm", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Active-record ORM"},
	{Name: "view", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "HTML view engine"},
	{Name: "queue", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Job queues"},
	{Name: "events", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Event dispatcher"},
	{Name: "localization", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Translator / locales"},
	{Name: "schedule", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Task scheduler"},
	{Name: "filesystem", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Filesystem disks"},
	{Name: "notification", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Async multi-channel notifications (mail, SMS, push, database, broadcast)"},
	{Name: "broadcasting", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Event broadcasting"},
	{Name: "httpclient", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Outbound HTTP client"},
	{Name: "ratelimit", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Rate limiter"},
	{Name: "url", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "URL generator"},
	{Name: "health", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Health checks"},
	{Name: "observability", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Metrics collector"},
	{Name: "maintenance", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Maintenance mode"},
	{Name: "apitoken", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Personal access tokens"},
	{Name: "assets", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "Asset manifest / Vite mix"},
	{Name: "console", Layer: kernel.LayerFoundation, Kind: kernel.KindService, Description: "CLI application"},

	{Name: "ai", Layer: kernel.LayerIntelligence, Kind: kernel.KindService, Stability: "experimental", Description: "AI chat providers"},
	{Name: "rag", Layer: kernel.LayerIntelligence, Kind: kernel.KindLibrary, Stability: "experimental", Description: "RAG chunking, embed pipeline, vector store helpers"},
	{Name: "agent", Layer: kernel.LayerIntelligence, Kind: kernel.KindLibrary, Stability: "experimental", Description: "AI agent loop, tools, conversation memory"},
	{Name: "workflow", Layer: kernel.LayerIntelligence, Kind: kernel.KindLibrary, Stability: "experimental", Description: "Generic process graphs (agents enter via agent.AsExecutor)"},

	{Name: "audit", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Request/audit event log"},
	{Name: "backup", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Database backup/restore (SQLite + native dump tools)"},
	{Name: "docs", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Markdown docs repository"},
	{Name: "graphql", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "GraphQL schema and queries"},
	{Name: "mongo", Layer: kernel.LayerAddon, Kind: kernel.KindService, Heavy: true, Description: "MongoDB client (separate module)"},
	{Name: "oauth", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "OAuth2 authorization server"},
	{Name: "seo", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Classic crawler SEO and LLM discovery"},
	{Name: "social", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Social OAuth login (GitHub/Google)"},
	{Name: "webauthn", Layer: kernel.LayerAddon, Kind: kernel.KindService, Heavy: true, Description: "WebAuthn/passkeys (separate module)"},
	{Name: "webhooks", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Outbound signed webhooks"},

	{Name: "api", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "API versioning helpers"},
	{Name: "browser", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Headless browser testing"},
	{Name: "consent", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Cookie/consent helpers"},
	{Name: "export", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "CSV/XLSX import and export"},
	{Name: "factory", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Model factories"},
	{Name: "fingerprint", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Device fingerprinting"},
	{Name: "honeypot", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Spam honeypot fields"},
	{Name: "idempotency", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Idempotency keys"},
	{Name: "image", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Image processing helpers"},
	{Name: "jsonapi", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "JSON:API document helpers"},
	{Name: "negotiate", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Content negotiation"},
	{Name: "openapi", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "OpenAPI generator helpers"},
	{Name: "pages", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Static page helpers"},
	{Name: "pdf", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "PDF generation and inline viewing"},
	{Name: "qr", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Heavy: true, Description: "QR code generation"},
	{Name: "resources", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "API resource transformers"},
	{Name: "testing", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Test helpers"},
	{Name: "toolkit/arr", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Array/slice helpers"},
	{Name: "toolkit/bloom", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Bloom filter"},
	{Name: "toolkit/circuit", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Circuit breaker"},
	{Name: "toolkit/collection", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Collection helpers"},
	{Name: "toolkit/color", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Color helpers"},
	{Name: "toolkit/concurrency", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Concurrency primitives"},
	{Name: "toolkit/cron", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Cron expression parser"},
	{Name: "toolkit/date", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Date/time helpers"},
	{Name: "toolkit/debug", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Debug dump helpers"},
	{Name: "toolkit/enums", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "String-backed enums with labels"},
	{Name: "toolkit/hashid", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Obfuscated public IDs"},
	{Name: "toolkit/html", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "HTML helpers"},
	{Name: "toolkit/jsonschema", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "JSON Schema validation"},
	{Name: "toolkit/lock", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Process-local atomic locks"},
	{Name: "toolkit/markdown", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Markdown renderer"},
	{Name: "toolkit/money", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Money helpers"},
	{Name: "toolkit/num", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Number helpers"},
	{Name: "toolkit/process", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "OS process runner"},
	{Name: "toolkit/str", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "String helpers"},
	{Name: "toolkit/timing", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Timing / stopwatch helpers"},
	{Name: "toolkit/zip", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "ZIP archive helpers"},
	{Name: "websocket", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "WebSocket helpers"},
}

func catalogAll() []kernel.PackageInfo {
	out := make([]kernel.PackageInfo, 0, len(kernel.Catalog)+len(ecosystemCatalog))
	out = append(out, kernel.Catalog...)
	out = append(out, ecosystemCatalog...)
	return out
}

func catalogLookup(name string) (kernel.PackageInfo, bool) {
	if p, ok := kernel.LookupPackage(name); ok {
		return p, true
	}
	for _, p := range ecosystemCatalog {
		if p.Name == name {
			return p, true
		}
	}
	return kernel.PackageInfo{}, false
}

func catalogByLayer(layer kernel.Layer) []kernel.PackageInfo {
	out := make([]kernel.PackageInfo, 0)
	for _, p := range catalogAll() {
		if p.Layer == layer {
			out = append(out, p)
		}
	}
	return out
}

func catalogLibraries() []kernel.PackageInfo {
	out := make([]kernel.PackageInfo, 0)
	for _, p := range catalogAll() {
		if p.Layer == kernel.LayerPrimitive {
			continue
		}
		if p.EffectiveKind() == kernel.KindLibrary {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
