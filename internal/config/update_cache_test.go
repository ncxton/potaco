package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUpdateCacheMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".potaco.json")

	cache, err := LoadUpdateCache(path)
	if err != nil {
		t.Fatalf("LoadUpdateCache missing file: %v", err)
	}
	if !cache.LastUpdateCheck.IsZero() {
		t.Fatalf("LastUpdateCheck = %v, want zero", cache.LastUpdateCheck)
	}
	if cache.LatestVersion != "" || cache.DismissedVersion != "" {
		t.Fatalf("cache = %+v, want empty", cache)
	}
}

func TestUpdateCacheRoundTripPrivateMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".potaco", ".potaco.json")
	want := &UpdateCache{
		LastUpdateCheck:  time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC),
		LatestVersion:    "v1.2.3",
		DismissedVersion: "v1.2.2",
	}

	if err := SaveUpdateCache(path, want); err != nil {
		t.Fatalf("SaveUpdateCache: %v", err)
	}

	got, err := LoadUpdateCache(path)
	if err != nil {
		t.Fatalf("LoadUpdateCache: %v", err)
	}
	if !got.LastUpdateCheck.Equal(want.LastUpdateCheck) {
		t.Fatalf("LastUpdateCheck = %v, want %v", got.LastUpdateCheck, want.LastUpdateCheck)
	}
	if got.LatestVersion != want.LatestVersion || got.DismissedVersion != want.DismissedVersion {
		t.Fatalf("cache = %+v, want %+v", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat cache: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("cache mode = %o, want 0600", got)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat cache dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Fatalf("cache dir mode = %o, want 0700", got)
	}
}

func TestSaveUpdateCacheDoesNotLeaveTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".potaco.json")
	if err := SaveUpdateCache(path, &UpdateCache{LatestVersion: "v1.2.3"}); err != nil {
		t.Fatalf("SaveUpdateCache: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".potaco-") && strings.HasSuffix(entry.Name(), ".tmp") {
			t.Fatalf("temporary cache file left behind: %s", entry.Name())
		}
	}
}

func TestDefaultUpdateCachePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := DefaultUpdateCachePath()
	want := filepath.Join(home, ".potaco", ".potaco.json")
	if got != want {
		t.Fatalf("DefaultUpdateCachePath = %q, want %q", got, want)
	}
}

func TestLoadUpdateCacheInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".potaco.json")
	if err := os.WriteFile(path, []byte("{"), 0600); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	_, err := LoadUpdateCache(path)
	if err == nil {
		t.Fatal("LoadUpdateCache should reject invalid JSON")
	}
	if errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("invalid JSON should not look like missing file: %v", err)
	}
}
