package bootstrap

// Lean production presets for App() / package:preset.
// These are suggestions — copy into EnabledAddons or apply via CLI.

// PresetAPI is a lean JSON/API-oriented addon set (no heavy modules).
var PresetAPI = []string{
	"circuit",
	"features",
	"hashid",
	"lock",
	"otp",
	"webhooks",
	"wellknown",
}

// PresetWeb is a lean public-web addon set (SEO/security basics, no heavy modules).
var PresetWeb = []string{
	"audit",
	"features",
	"hashid",
	"sitemap",
	"wellknown",
}

// PresetNames lists known preset identifiers for CLI discovery.
func PresetNames() []string {
	return []string{"api", "web", "demo"}
}

// Preset returns a named addon list (api|web|demo).
func Preset(name string) ([]string, bool) {
	switch name {
	case "api":
		return append([]string(nil), PresetAPI...), true
	case "web":
		return append([]string(nil), PresetWeb...), true
	case "demo":
		return append([]string(nil), DemoAddons...), true
	default:
		return nil, false
	}
}
