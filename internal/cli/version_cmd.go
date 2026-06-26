package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// githubReleaseURL is the GitHub API endpoint for the latest release.
// It is a package-level variable so tests can override it.
var githubReleaseURL = "https://api.github.com/repos/ncxton/potaco/releases/latest"

// Cache fields for checkLatestVersion. Cached for latestCacheTTL so
// repeated calls within the same process do not hit the API multiple
// times (e.g. version command + update command sharing the cache).
var (
	latestCache     string
	latestCacheTime time.Time
	latestCacheErr  error
)

const latestCacheTTL = 1 * time.Hour

// checkLatestVersion queries the GitHub API for the latest release tag.
// The result is cached for latestCacheTTL. On error, returns ("", err)
// without caching the error so a subsequent call can retry.
func checkLatestVersion() (string, error) {
	if time.Since(latestCacheTime) < latestCacheTTL && latestCache != "" {
		return latestCache, nil
	}

	resp, err := http.Get(githubReleaseURL)
	if err != nil {
		return "", fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("parse release JSON: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("github API returned empty tag_name")
	}

	// Cache the successful result.
	latestCache = release.TagName
	latestCacheTime = time.Now()

	return release.TagName, nil
}
