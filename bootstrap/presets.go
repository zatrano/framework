package bootstrap

// Lean production presets for App() / package:preset.
// Addon names here must be blank-imported by the consumer (github.com/zatrano/packages).
// This framework repo only ships the intelligence service `ai`.

// PresetAPI is a lean JSON/API-oriented addon set (no heavy modules).
var PresetAPI = []string{}

// PresetWeb is a lean public-web addon set (SEO/security basics, no heavy modules).
var PresetWeb = []string{}

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
