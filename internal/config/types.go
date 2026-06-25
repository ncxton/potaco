package config

import "time"

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
