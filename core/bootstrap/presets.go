package bootstrap

// PresetAPI / PresetWeb remain as empty name lists for CLI discovery.
var PresetAPI = []string{}
var PresetWeb = []string{}

func PresetNames() []string {
	return []string{"api", "web"}
}

func Preset(name string) ([]string, bool) {
	switch name {
	case "api":
		return append([]string(nil), PresetAPI...), true
	case "web":
		return append([]string(nil), PresetWeb...), true
	default:
		return nil, false
	}
}

// WithPresetAPI is App() plus any names in PresetAPI that are blank-imported.
func WithPresetAPI() Option { return WithAddons(PresetAPI...) }

// WithPresetWeb is App() plus any names in PresetWeb that are blank-imported.
func WithPresetWeb() Option { return WithAddons(PresetWeb...) }
