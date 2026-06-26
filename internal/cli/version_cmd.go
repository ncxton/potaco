package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
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

	latestCache = release.TagName
	latestCacheTime = time.Now()

	return release.TagName, nil
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current version and check for updates",
	RunE:  runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) error {
	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	out := cmd.OutOrStdout()

	latest, latestErr := checkLatestVersion()

	updateAvailable := false
	if latestErr == nil && latest != "" && latest != Version {
		updateAvailable = true
	}

	if jsonMode {
		type versionJSON struct {
			Current         string `json:"current"`
			Latest          string `json:"latest"`
			UpdateAvailable bool   `json:"update_available"`
		}
		vj := versionJSON{
			Current:         Version,
			Latest:          latest,
			UpdateAvailable: updateAvailable,
		}
		data, err := json.MarshalIndent(vj, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		fmt.Fprintln(out, string(data))
		return nil
	}

	fmt.Fprintf(out, "potaco %s", Version)

	if latestErr == nil && latest != "" {
		if updateAvailable {
			fmt.Fprintf(out, " (latest: %s, update available)\n", latest)
		} else {
			fmt.Fprintf(out, " (latest: %s, up to date)\n", latest)
		}
	} else {
		fmt.Fprintln(out)
	}

	return nil
}
