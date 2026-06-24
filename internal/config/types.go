package config

import "time"

// Config holds the resolved provider configuration after merging
// all precedence layers (flags, env, config file, presets).
type Config struct {
	BaseURL  string
	APIKey   string
	Model    string
	Retries  int
	Timeout  time.Duration
	Size     string
	Quality  string
	Provider string // preset name if specified
}

// ProviderPreset describes a named provider configuration preset.
type ProviderPreset struct {
	BaseURL      string   `yaml:"base_url"`
	DefaultModel string   `yaml:"default_model"`
	Sizes        []string `yaml:"sizes"`
}

// FileConfig represents the raw YAML structure of ~/.potaco/config.yaml.
type FileConfig struct {
	Default struct {
		BaseURL string `yaml:"base_url"`
		APIKey  string `yaml:"api_key"`
		Model   string `yaml:"model"`
		Retries int    `yaml:"retries"`
		Timeout string `yaml:"timeout"`
	} `yaml:"default"`
}

// MergeOptions holds optional CLI flag values for the merge.
// Only non-nil fields override lower-precedence sources.
type MergeOptions struct {
	BaseURL  *string
	APIKey   *string
	Model    *string
	Retries  *int
	Timeout  *time.Duration
	Provider *string
}
