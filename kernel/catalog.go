package kernel

// Layer classifies a framework package for the packages migration.
// Primitive boots always; Foundation is opt-in but common; Intelligence is the
// first-party AI identity layer; Addon is project opt-in.
type Layer string

const (
	// LayerPrimitive is always on the boot path: process must start; secure HTTP surface.
	LayerPrimitive Layer = "primitive"
	// LayerFoundation is opt-in but common: typical web/API application services.
	LayerFoundation Layer = "foundation"
	// LayerIntelligence is the first-party AI identity layer (same activation as addons).
	LayerIntelligence Layer = "intelligence"
	// LayerAddon is project opt-in: services and libraries enabled by the consumer.
	LayerAddon Layer = "addon"
)

// Kind classifies how an addon is consumed.
type Kind string

const (
	// KindService needs a provider / container binding (package:enable).
	KindService Kind = "service"
	// KindLibrary is import-only (no boot wiring).
	KindLibrary Kind = "library"
)

// PackageInfo describes one first-party package.
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

// Catalog is the kernel's own primitive surface. Foundation, intelligence, and
// addon names live in the CLI aggregator (console) so the kernel does not know
// "auth", "billing", or "agent".
var Catalog = []PackageInfo{
	{Name: "container", Layer: LayerPrimitive, Description: "Service container"},
	{Name: "config", Layer: LayerPrimitive, Description: "Configuration repository"},
	{Name: "env", Layer: LayerPrimitive, Description: "Environment loader"},
	{Name: "context", Layer: LayerPrimitive, Description: "Request/app context store"},
	{Name: "http", Layer: LayerPrimitive, Description: "HTTP request/response helpers"},
	{Name: "routing", Layer: LayerPrimitive, Description: "HTTP router"},
	{Name: "middleware", Layer: LayerPrimitive, Description: "HTTP middleware primitives"},
	{Name: "pipeline", Layer: LayerPrimitive, Description: "Middleware pipeline"},
	{Name: "exceptions", Layer: LayerPrimitive, Description: "Exception handler"},
	{Name: "log", Layer: LayerPrimitive, Description: "Application logger"},
	{Name: "encryption", Layer: LayerPrimitive, Description: "Symmetric encryption"},
	{Name: "trustedproxy", Layer: LayerPrimitive, Description: "Trusted proxy headers"},
	{Name: "report", Layer: LayerPrimitive, Description: "Exception reporting"},
	{Name: "cookie", Layer: LayerPrimitive, Description: "Cookie jar helpers"},
	{Name: "support", Layer: LayerPrimitive, Description: "Support helpers"},
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

// LookupPackage finds a kernel catalog entry by name (primitives only).
func LookupPackage(name string) (PackageInfo, bool) {
	for _, p := range Catalog {
		if p.Name == name {
			return p, true
		}
	}
	return PackageInfo{}, false
}
