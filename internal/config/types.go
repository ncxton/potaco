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
	Provider string // preset name if specified
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

// ProviderConfig holds per-provider settings in the multi-provider config format.
type ProviderConfig struct {
	Model   string        `yaml:"model"`
	Retries int           `yaml:"retries"`
	Timeout time.Duration `yaml:"timeout"`
}

// MultiProviderConfig is the v2 config format supporting multiple providers
// with separate credentials and an active provider/model selector.
type MultiProviderConfig struct {
	ActiveProvider string                    `yaml:"active_provider"`
	ActiveModel    string                    `yaml:"active_model"`
	Providers      map[string]ProviderConfig `yaml:"providers"`
}
