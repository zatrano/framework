package console

import "github.com/zatrano/framework/v2/kernel"

// ecosystemCatalog is the CLI aggregation of packages-module names.
// The kernel catalog stays primitive-only; this list is how package:list,
// doctor, and describe know foundation / intelligence / addon packages
// without the kernel importing them.
var ecosystemCatalog = []kernel.PackageInfo{
	{Name: "session", Layer: kernel.LayerFoundation, Description: "HTTP sessions"},
	{Name: "flash", Layer: kernel.LayerFoundation, Description: "Flash / toast messages"},
	{Name: "validation", Layer: kernel.LayerFoundation, Description: "Input validation"},
	{Name: "auth", Layer: kernel.LayerFoundation, Description: "Authentication guards"},
	{Name: "authorization", Layer: kernel.LayerFoundation, Description: "Gates and policies"},
	{Name: "hashing", Layer: kernel.LayerFoundation, Description: "Password hashing"},
	{Name: "cache", Layer: kernel.LayerFoundation, Description: "Cache manager"},
	{Name: "redisx", Layer: kernel.LayerFoundation, Description: "Redis client helper"},
	{Name: "database", Layer: kernel.LayerFoundation, Description: "Database manager"},
	{Name: "orm", Layer: kernel.LayerFoundation, Description: "Active-record ORM"},
	{Name: "view", Layer: kernel.LayerFoundation, Description: "HTML view engine"},
	{Name: "queue", Layer: kernel.LayerFoundation, Description: "Job queues"},
	{Name: "events", Layer: kernel.LayerFoundation, Description: "Event dispatcher"},
	{Name: "localization", Layer: kernel.LayerFoundation, Description: "Translator / locales"},
	{Name: "schedule", Layer: kernel.LayerFoundation, Description: "Task scheduler"},
	{Name: "filesystem", Layer: kernel.LayerFoundation, Description: "Filesystem disks"},
	{Name: "notification", Layer: kernel.LayerFoundation, Description: "Async multi-channel notifications (mail, SMS, push, database, broadcast)"},
	{Name: "broadcasting", Layer: kernel.LayerFoundation, Description: "Event broadcasting"},
	{Name: "httpclient", Layer: kernel.LayerFoundation, Description: "Outbound HTTP client"},
	{Name: "ratelimit", Layer: kernel.LayerFoundation, Description: "Rate limiter"},
	{Name: "url", Layer: kernel.LayerFoundation, Description: "URL generator"},
	{Name: "health", Layer: kernel.LayerFoundation, Description: "Health checks"},
	{Name: "observability", Layer: kernel.LayerFoundation, Description: "Metrics collector"},
	{Name: "maintenance", Layer: kernel.LayerFoundation, Description: "Maintenance mode"},
	{Name: "apitoken", Layer: kernel.LayerFoundation, Description: "Personal access tokens"},
	{Name: "assets", Layer: kernel.LayerFoundation, Description: "Asset manifest / Vite mix"},
	{Name: "console", Layer: kernel.LayerFoundation, Description: "CLI application"},
	{Name: "version", Layer: kernel.LayerFoundation, Description: "Version helper"},

	{Name: "ai", Layer: kernel.LayerIntelligence, Kind: kernel.KindService, Description: "AI chat providers"},
	{Name: "rag", Layer: kernel.LayerIntelligence, Kind: kernel.KindLibrary, Description: "RAG chunking, embed pipeline, vector store helpers"},
	{Name: "agent", Layer: kernel.LayerIntelligence, Kind: kernel.KindLibrary, Description: "AI agent loop, tools, conversation memory"},

	{Name: "audit", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Request/audit event log"},
	{Name: "backup", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Database backup/restore (SQLite + native dump tools)"},
	{Name: "billing", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Central billing manager (memory/stripe gateways, webhooks)"},
	{Name: "bus", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Synchronous command bus"},
	{Name: "circuit", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Circuit breaker"},
	{Name: "docs", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Markdown docs repository"},
	{Name: "enums", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "String enum registry"},
	{Name: "features", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Feature flags and gradual rollouts"},
	{Name: "geo", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Geolocation resolver"},
	{Name: "graphql", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "GraphQL schema and queries"},
	{Name: "hashid", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Obfuscated public IDs"},
	{Name: "inspector", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Request inspector toolbar data"},
	{Name: "lock", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Atomic locks"},
	{Name: "mongo", Layer: kernel.LayerAddon, Kind: kernel.KindService, Heavy: true, Description: "MongoDB client (separate module)"},
	{Name: "oauth", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "OAuth2 authorization server"},
	{Name: "octane", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Concurrent runtime metrics"},
	{Name: "otp", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "One-time passwords"},
	{Name: "pulse", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Metrics pulse dashboard"},
	{Name: "search", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "In-memory search engine"},
	{Name: "shorturl", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Short URL manager"},
	{Name: "sitemap", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "XML sitemap builder"},
	{Name: "social", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Social OAuth login (GitHub/Google)"},
	{Name: "tenancy", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Multi-tenant resolution"},
	{Name: "webauthn", Layer: kernel.LayerAddon, Kind: kernel.KindService, Heavy: true, Description: "WebAuthn/passkeys (separate module)"},
	{Name: "webhooks", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "Outbound signed webhooks"},
	{Name: "wellknown", Layer: kernel.LayerAddon, Kind: kernel.KindService, Description: "security.txt / well-known"},

	{Name: "api", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "API versioning helpers"},
	{Name: "archive", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "ZIP archive helpers"},
	{Name: "bloom", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Bloom filter"},
	{Name: "browser", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Headless browser testing"},
	{Name: "collection", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Collection helpers"},
	{Name: "concurrency", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Concurrency primitives"},
	{Name: "consent", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Cookie/consent helpers"},
	{Name: "cron", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Cron expression parser"},
	{Name: "debug", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Debug dump helpers"},
	{Name: "export", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "CSV/XLSX import and export"},
	{Name: "factory", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Model factories"},
	{Name: "fingerprint", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Device fingerprinting"},
	{Name: "honeypot", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Spam honeypot fields"},
	{Name: "idempotency", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Idempotency keys"},
	{Name: "image", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Image processing helpers"},
	{Name: "jsonapi", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "JSON:API document helpers"},
	{Name: "jsonschema", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "JSON Schema validation"},
	{Name: "markdown", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Markdown renderer"},
	{Name: "negotiate", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Content negotiation"},
	{Name: "openapi", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "OpenAPI generator helpers"},
	{Name: "pages", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Static page helpers"},
	{Name: "pagination", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Paginator helpers"},
	{Name: "pdf", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "PDF generation and inline viewing"},
	{Name: "process", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "OS process runner"},
	{Name: "qr", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Heavy: true, Description: "QR code generation"},
	{Name: "resources", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "API resource transformers"},
	{Name: "testing", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Test helpers"},
	{Name: "timing", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "Timing / stopwatch helpers"},
	{Name: "totp", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "TOTP codes"},
	{Name: "useragent", Layer: kernel.LayerAddon, Kind: kernel.KindLibrary, Description: "User-Agent parser"},
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
	return out
}
