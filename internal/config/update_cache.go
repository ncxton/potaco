package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type UpdateCache struct {
	LastUpdateCheck  time.Time `json:"last_update_check"`
	LatestVersion    string    `json:"latest_version"`
	DismissedVersion string    `json:"dismissed_version"`
}

func DefaultUpdateCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".potaco", ".potaco.json")
	}
	return filepath.Join(home, ".potaco", ".potaco.json")
}

func LoadUpdateCache(path string) (*UpdateCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &UpdateCache{}, nil
		}
		return nil, fmt.Errorf("read update cache: %w", err)
	}
	var cache UpdateCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parse update cache: %w", err)
	}
	return &cache, nil
}

func SaveUpdateCache(path string, cache *UpdateCache) error {
	if cache == nil {
		cache = &UpdateCache{}
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create update cache dir: %w", err)
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal update cache: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".potaco-*.tmp")
	if err != nil {
		return fmt.Errorf("create update cache temp: %w", err)
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
		return fmt.Errorf("set update cache temp perms: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write update cache temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close update cache temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("write update cache: %w", err)
	}
	cleanup = false
	return nil
}
