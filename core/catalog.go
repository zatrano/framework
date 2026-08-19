package core

// Layer classifies a framework package for the packages migration.
// Kernel boots always; Foundation is opt-in but common; Addon is project opt-in.
type Layer string

const (
	LayerKernel     Layer = "kernel"
	LayerFoundation Layer = "foundation"
	LayerAddon      Layer = "addon"
)

// Kind classifies how an addon is consumed.
type Kind string

const (
	// KindService needs a provider / container binding (package:enable).
	KindService Kind = "service"
	// KindLibrary is import-only (no boot wiring).
	KindLibrary Kind = "library"
)

// PackageInfo describes one first-party package under packages/.
type PackageInfo struct {
	Name        string
	Layer       Layer
	Kind        Kind
	Heavy       bool
	Description string
}

// EffectiveKind returns the consumption kind for this package.
func (p PackageInfo) EffectiveKind() Kind {
	if p.Kind != "" {
		return p.Kind
	}
	if p.Layer == LayerAddon {
		return KindLibrary
	}
	return KindService
}

// Catalog is the source of truth for package layering.
// Kernel lives in thin core/; implementations live under packages/.
var Catalog = []PackageInfo{
	// Kernel — process must start; secure HTTP surface.
	{Name: "container", Layer: LayerKernel, Description: "Service container"},
	{Name: "config", Layer: LayerKernel, Description: "Configuration repository"},
	{Name: "env", Layer: LayerKernel, Description: "Environment loader"},
	{Name: "context", Layer: LayerKernel, Description: "Request/app context store"},
	{Name: "http", Layer: LayerKernel, Description: "HTTP request/response helpers"},
	{Name: "routing", Layer: LayerKernel, Description: "HTTP router"},
	{Name: "middleware", Layer: LayerKernel, Description: "HTTP middleware primitives"},
	{Name: "pipeline", Layer: LayerKernel, Description: "Middleware pipeline"},
	{Name: "exceptions", Layer: LayerKernel, Description: "Exception handler"},
	{Name: "log", Layer: LayerKernel, Description: "Application logger"},
	{Name: "encryption", Layer: LayerKernel, Description: "Symmetric encryption"},
	{Name: "trustedproxy", Layer: LayerKernel, Description: "Trusted proxy headers"},
	{Name: "report", Layer: LayerKernel, Description: "Exception reporting"},

	// Foundation — typical web/API apps.
	{Name: "session", Layer: LayerFoundation, Description: "HTTP sessions"},
	{Name: "cookie", Layer: LayerFoundation, Description: "Cookie jar helpers"},
	{Name: "flash", Layer: LayerFoundation, Description: "Flash / toast messages"},
	{Name: "validation", Layer: LayerFoundation, Description: "Input validation"},
	{Name: "auth", Layer: LayerFoundation, Description: "Authentication guards"},
	{Name: "authorization", Layer: LayerFoundation, Description: "Gates and policies"},
	{Name: "hashing", Layer: LayerFoundation, Description: "Password hashing"},
	{Name: "cache", Layer: LayerFoundation, Description: "Cache manager"},
	{Name: "redisx", Layer: LayerFoundation, Description: "Redis client helper"},
	{Name: "database", Layer: LayerFoundation, Description: "Database manager"},
	{Name: "orm", Layer: LayerFoundation, Description: "Active-record ORM"},
	{Name: "view", Layer: LayerFoundation, Description: "HTML view engine"},
	{Name: "mail", Layer: LayerFoundation, Description: "Mail manager"},
	{Name: "queue", Layer: LayerFoundation, Description: "Job queues"},
	{Name: "events", Layer: LayerFoundation, Description: "Event dispatcher"},
	{Name: "localization", Layer: LayerFoundation, Description: "Translator / locales"},
	{Name: "schedule", Layer: LayerFoundation, Description: "Task scheduler"},
	{Name: "filesystem", Layer: LayerFoundation, Description: "Filesystem disks"},
	{Name: "notification", Layer: LayerFoundation, Description: "Multi-channel notifications"},
	{Name: "broadcasting", Layer: LayerFoundation, Description: "Event broadcasting"},
	{Name: "httpclient", Layer: LayerFoundation, Description: "Outbound HTTP client"},
	{Name: "ratelimit", Layer: LayerFoundation, Description: "Rate limiter"},
	{Name: "url", Layer: LayerFoundation, Description: "URL generator"},
	{Name: "health", Layer: LayerFoundation, Description: "Health checks"},
	{Name: "observability", Layer: LayerFoundation, Description: "Metrics collector"},
	{Name: "maintenance", Layer: LayerFoundation, Description: "Maintenance mode"},
	{Name: "apitoken", Layer: LayerFoundation, Description: "Personal access tokens"},
	{Name: "assets", Layer: LayerFoundation, Description: "Asset manifest / Vite mix"},
	{Name: "console", Layer: LayerFoundation, Description: "CLI application"},
	{Name: "support", Layer: LayerFoundation, Description: "Support helpers"},
	{Name: "version", Layer: LayerFoundation, Description: "Version helper"},

	// Addon services — opt-in via bootstrap/addons registry + EnabledAddons.
	{Name: "ai", Layer: LayerAddon, Kind: KindService, Description: "AI chat providers"},
	{Name: "audit", Layer: LayerAddon, Kind: KindService, Description: "Request/audit event log"},
	{Name: "backup", Layer: LayerAddon, Kind: KindService, Description: "SQLite/database backup helper"},
	{Name: "billing", Layer: LayerAddon, Kind: KindService, Description: "Billing/checkout helpers"},
	{Name: "bus", Layer: LayerAddon, Kind: KindService, Description: "Synchronous command bus"},
	{Name: "circuit", Layer: LayerAddon, Kind: KindService, Description: "Circuit breaker"},
	{Name: "docs", Layer: LayerAddon, Kind: KindService, Description: "Markdown docs repository"},
	{Name: "enums", Layer: LayerAddon, Kind: KindService, Description: "String enum registry"},
	{Name: "features", Layer: LayerAddon, Kind: KindService, Description: "Feature flags and gradual rollouts"},
	{Name: "geo", Layer: LayerAddon, Kind: KindService, Description: "Geolocation resolver"},
	{Name: "graphql", Layer: LayerAddon, Kind: KindService, Description: "GraphQL schema and queries"},
	{Name: "hashid", Layer: LayerAddon, Kind: KindService, Description: "Obfuscated public IDs"},
	{Name: "inspector", Layer: LayerAddon, Kind: KindService, Description: "Request inspector toolbar data"},
	{Name: "lock", Layer: LayerAddon, Kind: KindService, Description: "Atomic locks"},
	{Name: "mongo", Layer: LayerAddon, Kind: KindService, Heavy: true, Description: "MongoDB client (separate module)"},
	{Name: "oauth", Layer: LayerAddon, Kind: KindService, Description: "OAuth2 authorization server"},
	{Name: "octane", Layer: LayerAddon, Kind: KindService, Description: "Concurrent runtime metrics"},
	{Name: "otp", Layer: LayerAddon, Kind: KindService, Description: "One-time passwords"},
	{Name: "pulse", Layer: LayerAddon, Kind: KindService, Description: "Metrics pulse dashboard"},
	{Name: "search", Layer: LayerAddon, Kind: KindService, Description: "In-memory search engine"},
	{Name: "shorturl", Layer: LayerAddon, Kind: KindService, Description: "Short URL manager"},
	{Name: "sitemap", Layer: LayerAddon, Kind: KindService, Description: "XML sitemap builder"},
	{Name: "social", Layer: LayerAddon, Kind: KindService, Description: "Social OAuth login (GitHub/Google)"},
	{Name: "tenancy", Layer: LayerAddon, Kind: KindService, Description: "Multi-tenant resolution"},
	{Name: "webauthn", Layer: LayerAddon, Kind: KindService, Heavy: true, Description: "WebAuthn/passkeys (separate module)"},
	{Name: "webhooks", Layer: LayerAddon, Kind: KindService, Description: "Outbound signed webhooks"},
	{Name: "wellknown", Layer: LayerAddon, Kind: KindService, Description: "security.txt / well-known"},

	// Addon libraries — import and use; no package:enable / container binding.
	{Name: "api", Layer: LayerAddon, Kind: KindLibrary, Description: "API versioning helpers"},
	{Name: "archive", Layer: LayerAddon, Kind: KindLibrary, Description: "ZIP archive helpers"},
	{Name: "bloom", Layer: LayerAddon, Kind: KindLibrary, Description: "Bloom filter"},
	{Name: "browser", Layer: LayerAddon, Kind: KindLibrary, Description: "Headless browser testing"},
	{Name: "collection", Layer: LayerAddon, Kind: KindLibrary, Description: "Collection helpers"},
	{Name: "concurrency", Layer: LayerAddon, Kind: KindLibrary, Description: "Concurrency primitives"},
	{Name: "consent", Layer: LayerAddon, Kind: KindLibrary, Description: "Cookie/consent helpers"},
	{Name: "cron", Layer: LayerAddon, Kind: KindLibrary, Description: "Cron expression parser"},
	{Name: "debug", Layer: LayerAddon, Kind: KindLibrary, Description: "Debug dump helpers"},
	{Name: "export", Layer: LayerAddon, Kind: KindLibrary, Description: "CSV/XLSX import and export"},
	{Name: "factory", Layer: LayerAddon, Kind: KindLibrary, Description: "Model factories"},
	{Name: "fingerprint", Layer: LayerAddon, Kind: KindLibrary, Description: "Device fingerprinting"},
	{Name: "honeypot", Layer: LayerAddon, Kind: KindLibrary, Description: "Spam honeypot fields"},
	{Name: "idempotency", Layer: LayerAddon, Kind: KindLibrary, Description: "Idempotency keys"},
	{Name: "image", Layer: LayerAddon, Kind: KindLibrary, Description: "Image processing helpers"},
	{Name: "jsonapi", Layer: LayerAddon, Kind: KindLibrary, Description: "JSON:API document helpers"},
	{Name: "jsonschema", Layer: LayerAddon, Kind: KindLibrary, Description: "JSON Schema validation"},
	{Name: "markdown", Layer: LayerAddon, Kind: KindLibrary, Description: "Markdown renderer"},
	{Name: "negotiate", Layer: LayerAddon, Kind: KindLibrary, Description: "Content negotiation"},
	{Name: "openapi", Layer: LayerAddon, Kind: KindLibrary, Description: "OpenAPI generator helpers"},
	{Name: "pages", Layer: LayerAddon, Kind: KindLibrary, Description: "Static page helpers"},
	{Name: "pagination", Layer: LayerAddon, Kind: KindLibrary, Description: "Paginator helpers"},
	{Name: "pdf", Layer: LayerAddon, Kind: KindLibrary, Description: "PDF generation and inline viewing"},
	{Name: "process", Layer: LayerAddon, Kind: KindLibrary, Description: "OS process runner"},
	{Name: "qr", Layer: LayerAddon, Kind: KindLibrary, Heavy: true, Description: "QR code generation"},
	{Name: "resources", Layer: LayerAddon, Kind: KindLibrary, Description: "API resource transformers"},
	{Name: "testing", Layer: LayerAddon, Kind: KindLibrary, Description: "Test helpers"},
	{Name: "timing", Layer: LayerAddon, Kind: KindLibrary, Description: "Timing / stopwatch helpers"},
	{Name: "totp", Layer: LayerAddon, Kind: KindLibrary, Description: "TOTP codes"},
	{Name: "useragent", Layer: LayerAddon, Kind: KindLibrary, Description: "User-Agent parser"},
	{Name: "websocket", Layer: LayerAddon, Kind: KindLibrary, Description: "WebSocket helpers"},
}

// PackagesByLayer returns catalog entries for a layer.
func PackagesByLayer(layer Layer) []PackageInfo {
	out := make([]PackageInfo, 0)
	for _, p := range Catalog {
		if p.Layer == layer {
			out = append(out, p)
		}
	}
	return out
}

// PackagesByKind returns catalog entries with the given effective kind.
func PackagesByKind(kind Kind) []PackageInfo {
	out := make([]PackageInfo, 0)
	for _, p := range Catalog {
		if p.EffectiveKind() == kind {
			out = append(out, p)
		}
	}
	return out
}

// LookupPackage finds a catalog entry by name.
func LookupPackage(name string) (PackageInfo, bool) {
	for _, p := range Catalog {
		if p.Name == name {
			return p, true
		}
	}
	return PackageInfo{}, false
}
