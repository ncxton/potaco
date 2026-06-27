package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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
	migrateConfig(cfg)

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
