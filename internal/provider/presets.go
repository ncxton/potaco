package provider

// Preset holds known defaults for a specific provider.
type Preset struct {
	BaseURL      string
	DefaultModel string
	Sizes        []string
}

var presets = map[string]Preset{
	"openai": {
		BaseURL:      "https://api.openai.com",
		DefaultModel: "gpt-image-2",
		Sizes:        []string{"1024x1024", "1536x1024", "1024x1536"},
	},
	"together": {
		BaseURL:      "https://api.together.ai",
		DefaultModel: "black-forest-labs/flux-1",
		Sizes:        []string{"1024x1024"},
	},
	"fal": {
		BaseURL:      "https://fal.run",
		DefaultModel: "fal-ai/flux",
		Sizes:        []string{"1024x1024"},
	},
}

// GetPreset returns the preset for the named provider.
// Returns (Preset, true) if found, (Preset{}, false) otherwise.
func GetPreset(name string) (Preset, bool) {
	p, ok := presets[name]
	return p, ok
}

// AllPresets returns the full map of provider presets.
func AllPresets() map[string]Preset {
	return presets
}
