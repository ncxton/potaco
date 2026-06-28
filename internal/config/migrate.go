package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

type migrationFunc func(*MultiProviderConfig) bool

var migrations = map[int]migrationFunc{
	2: migrateProviderTypes,
}

func MigrateConfigFile(path string, now func() time.Time) (bool, string, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("stat config: %w", err)
	}

	cfg, err := LoadMultiProvider(path)
	if err != nil {
		return false, "", err
	}
	changed := migrateConfig(cfg)
	if !changed {
		return false, "", nil
	}

	backupPath, err := backupConfig(path, now)
	if err != nil {
		return false, "", err
	}
	if err := SaveMultiProvider(path, cfg); err != nil {
		return false, backupPath, err
	}
	return true, backupPath, nil
}

func migrateConfig(cfg *MultiProviderConfig) bool {
	changed := false
	for version := cfg.SchemaVersion + 1; version <= CurrentSchemaVersion; version++ {
		migrate, ok := migrations[version]
		if !ok {
			continue
		}
		if migrate(cfg) {
			changed = true
		}
		if cfg.SchemaVersion < version {
			cfg.SchemaVersion = version
			changed = true
		}
	}
	return changed
}

func migrateProviderTypes(cfg *MultiProviderConfig) bool {
	changed := false
	for name, pc := range cfg.Providers {
		if pc.Type != "" {
			continue
		}
		pc.Type = ResolveProviderType(name, pc)
		cfg.Providers[name] = pc
		changed = true
	}
	return changed
}

func ResolveProviderType(providerName string, pc ProviderConfig) string {
	if pc.Type != "" {
		return pc.Type
	}
	if providerName == "custom" {
		return "openai-compatible"
	}
	return providerName
}

func ProviderRequiresBaseURL(providerName, providerType string) bool {
	if providerType == "openai-compatible" || providerName == "custom" {
		return true
	}
	switch providerName {
	case "openai", "fal", "vercel":
		return false
	default:
		return true
	}
}

func AdapterType(providerType string) string {
	if providerType == "openai-compatible" {
		return "custom"
	}
	return providerType
}

func backupConfig(path string, now func() time.Time) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read config backup source: %w", err)
	}
	backupPath := fmt.Sprintf("%s.bak-%s", path, now().Format("20060102-150405"))
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return "", fmt.Errorf("write config backup: %w", err)
	}
	return backupPath, nil
}
