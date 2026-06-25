package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultConfigPath returns the default config file path at ~/.potaco/config.yaml.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".potaco/config.yaml"
	}
	return filepath.Join(home, ".potaco", "config.yaml")
}

// Load reads and parses a YAML config file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var fc FileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg := &Config{
		BaseURL: fc.Default.BaseURL,
		APIKey:  fc.Default.APIKey,
		Model:   fc.Default.Model,
		Retries: fc.Default.Retries,
	}

	if fc.Default.Retries == 0 {
		cfg.Retries = 2 // sensible default
	}

	if fc.Default.Timeout != "" {
		d, err := time.ParseDuration(fc.Default.Timeout)
		if err != nil {
			return nil, fmt.Errorf("parse timeout: %w", err)
		}
		cfg.Timeout = d
	} else {
		cfg.Timeout = 120 * time.Second
	}

	return cfg, nil
}

// FromEnv builds a Config from environment variables.
// Returns (nil, nil) if no env vars are set. Returns (nil, error) if a
// var is set but cannot be parsed, so the user learns the typo instead
// of silently falling back to defaults.
func FromEnv() (*Config, error) {
	cfg := &Config{}
	set := false

	if v := os.Getenv("POTACO_BASE_URL"); v != "" {
		cfg.BaseURL = v
		set = true
	}
	if v := os.Getenv("POTACO_API_KEY"); v != "" {
		cfg.APIKey = v
		set = true
	}
	if v := os.Getenv("POTACO_MODEL"); v != "" {
		cfg.Model = v
		set = true
	}
	if v := os.Getenv("POTACO_RETRIES"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("parse POTACO_RETRIES: %w", err)
		}
		cfg.Retries = n
		set = true
	}
	if v := os.Getenv("POTACO_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("parse POTACO_TIMEOUT: %w", err)
		}
		cfg.Timeout = d
		set = true
	}

	if !set {
		return nil, nil
	}
	if cfg.Retries == 0 {
		cfg.Retries = 2
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 120 * time.Second
	}
	return cfg, nil
}

// Merge resolves the final configuration by applying precedence:
// 1. CLI flags (non-nil fields in opts)
// 2. Environment variables
// 3. Config file (fileCfg, if non-nil)
// 4. Built-in defaults
func Merge(opts MergeOptions) (*Config, error) {
	fileCfg, err := loadFileConfig()
	if err != nil {
		return nil, err
	}
	return mergeInternal(opts, fileCfg)
}

// loadFileConfig reads the default config file. A missing file is not an
// error (return nil); a corrupted or unreadable file is, so the user
// learns their config is broken instead of silently falling back.
func loadFileConfig() (*Config, error) {
	cfg, err := Load(DefaultConfigPath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return cfg, nil
}

// mergeInternal is the testable core of Merge that accepts explicit inputs.
func mergeInternal(opts MergeOptions, fileCfg *Config) (*Config, error) {
	cfg := &Config{
		Retries: 2,
		Timeout: 120 * time.Second,
	}

	// Layer 3-4: file config
	if fileCfg != nil {
		cfg.BaseURL = fileCfg.BaseURL
		cfg.APIKey = fileCfg.APIKey
		cfg.Model = fileCfg.Model
		cfg.Retries = fileCfg.Retries
		cfg.Timeout = fileCfg.Timeout
	}

	// Layer 2: env vars (override file)
	envCfg, err := FromEnv()
	if err != nil {
		return nil, err
	}
	if envCfg != nil {
		if envCfg.BaseURL != "" {
			cfg.BaseURL = envCfg.BaseURL
		}
		if envCfg.APIKey != "" {
			cfg.APIKey = envCfg.APIKey
		}
		if envCfg.Model != "" {
			cfg.Model = envCfg.Model
		}
		if envCfg.Retries != 0 {
			cfg.Retries = envCfg.Retries
		}
		if envCfg.Timeout != 0 {
			cfg.Timeout = envCfg.Timeout
		}
	}

	// Layer 1: CLI flags (override everything)
	if opts.BaseURL != nil {
		cfg.BaseURL = *opts.BaseURL
	}
	if opts.APIKey != nil {
		cfg.APIKey = *opts.APIKey
	}
	if opts.Model != nil {
		cfg.Model = *opts.Model
	}
	if opts.Retries != nil {
		cfg.Retries = *opts.Retries
	}
	if opts.Timeout != nil {
		cfg.Timeout = *opts.Timeout
	}
	if opts.Provider != nil {
		cfg.Provider = *opts.Provider
	}

	// Validation
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("no base_url configured: set --base-url, POTACO_BASE_URL env, or config file default.base_url")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("no api_key configured: set --api-key, POTACO_API_KEY env, or config file default.api_key")
	}

	return cfg, nil
}

// LoadMultiProvider reads and parses a multi-provider config file.
// Returns an empty MultiProviderConfig (not nil, not an error) if the
// file does not exist.
func LoadMultiProvider(path string) (*MultiProviderConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &MultiProviderConfig{
				Providers: make(map[string]ProviderConfig),
			}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg MultiProviderConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}
	return &cfg, nil
}

// SaveMultiProvider writes the multi-provider config to the given path.
// The file is created with 0600 permissions and the directory with 0700.
func SaveMultiProvider(path string, cfg *MultiProviderConfig) error {
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("set temp config perms: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	cleanup = false
	return nil
}

// DefaultCredentialPath returns the default credential file path.
func DefaultCredentialPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".potaco/credentials.enc"
	}
	return filepath.Join(home, ".potaco", "credentials.enc")
}

// DefaultSaltPath returns the default salt file path.
func DefaultSaltPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".potaco/.salt"
	}
	return filepath.Join(home, ".potaco", ".salt")
}
