package config

const CurrentSchemaVersion = 2

// ProviderConfig holds per-provider settings in the multi-provider config format.
// Timeout is stored in seconds (plain integer in YAML) so users do not need
// to add unit suffixes like "s" or "m".
type ProviderConfig struct {
	Type    string `yaml:"type,omitempty"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url,omitempty"`
	Retries int    `yaml:"retries"`
	Timeout int    `yaml:"timeout"`
}

// MultiProviderConfig is the v2 config format supporting multiple providers
// with separate credentials and an active provider/model selector.
type MultiProviderConfig struct {
	SchemaVersion  int                       `yaml:"schema_version,omitempty"`
	ActiveProvider string                    `yaml:"active_provider"`
	ActiveModel    string                    `yaml:"active_model"`
	Providers      map[string]ProviderConfig `yaml:"providers"`
}
