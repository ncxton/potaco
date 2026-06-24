package config

import (
	"fmt"
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
// Returns nil if no env vars are set.
func FromEnv() *Config {
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
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Retries = n
			set = true
		}
	}
	if v := os.Getenv("POTACO_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout = d
			set = true
		}
	}

	if !set {
		return nil
	}
	if cfg.Retries == 0 {
		cfg.Retries = 2
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 120 * time.Second
	}
	return cfg
}

// Merge resolves the final configuration by applying precedence:
// 1. CLI flags (non-nil fields in opts)
// 2. Environment variables
// 3. Config file (fileCfg, if non-nil)
// 4. Built-in defaults
func Merge(opts MergeOptions) (*Config, error) {
	return mergeInternal(opts, loadFileConfig())
}

func loadFileConfig() *Config {
	cfg, err := Load(DefaultConfigPath())
	if err != nil {
		return nil
	}
	return cfg
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
	envCfg := FromEnv()
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
