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
