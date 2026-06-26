package config

// ProviderConfig holds per-provider settings in the multi-provider config format.
// Timeout is stored in seconds (plain integer in YAML) so users do not need
// to add unit suffixes like "s" or "m".
type ProviderConfig struct {
	Model   string `yaml:"model"`
	Retries int    `yaml:"retries"`
	Timeout int    `yaml:"timeout"` // seconds
}

// MultiProviderConfig is the v2 config format supporting multiple providers
// with separate credentials and an active provider/model selector.
type MultiProviderConfig struct {
	ActiveProvider string                    `yaml:"active_provider"`
	ActiveModel    string                    `yaml:"active_model"`
	Providers      map[string]ProviderConfig `yaml:"providers"`
}
