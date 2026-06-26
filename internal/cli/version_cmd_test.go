package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func resetVersionCache() {
	latestCache = ""
	latestCacheTime = time.Time{}
	latestCacheErr = nil
}

func TestCheckLatestVersion(t *testing.T) {
	resetVersionCache()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v9.9.9",
		})
	}))
	defer srv.Close()

	// Override the GitHub API URL
	origURL := githubReleaseURL
	githubReleaseURL = srv.URL
	defer func() { githubReleaseURL = origURL }()

	tag, err := checkLatestVersion()
	if err != nil {
		t.Fatalf("checkLatestVersion error: %v", err)
	}
	if tag != "v9.9.9" {
		t.Errorf("tag = %q, want %q", tag, "v9.9.9")
	}
}

func TestCheckLatestVersionCaches(t *testing.T) {
	resetVersionCache()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r
		calls++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v8.8.8",
		})
	}))
	defer srv.Close()

	origURL := githubReleaseURL
	githubReleaseURL = srv.URL
	defer func() { githubReleaseURL = origURL }()

	// First call hits the server
	tag, err := checkLatestVersion()
	if err != nil {
		t.Fatalf("first checkLatestVersion error: %v", err)
	}
	if tag != "v8.8.8" {
		t.Fatalf("first tag = %q, want %q", tag, "v8.8.8")
	}
	if calls != 1 {
		t.Fatalf("expected 1 server call, got %d", calls)
	}

	// Second call should use cache, no new server hit
	tag, err = checkLatestVersion()
	if err != nil {
		t.Fatalf("second checkLatestVersion error: %v", err)
	}
	if tag != "v8.8.8" {
		t.Fatalf("second tag = %q, want %q", tag, "v8.8.8")
	}
	if calls != 1 {
		t.Fatalf("expected 1 server call after cache, got %d", calls)
	}
}

func TestCheckLatestVersionFailsGracefully(t *testing.T) {
	resetVersionCache()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	origURL := githubReleaseURL
	githubReleaseURL = srv.URL
	defer func() { githubReleaseURL = origURL }()

	tag, err := checkLatestVersion()
	if err == nil {
		t.Fatal("expected error when server returns 500")
	}
	if tag != "" {
		t.Errorf("tag should be empty on error, got %q", tag)
	}
}

func TestVersionCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'version' subcommand")
	}
}

func TestVersionFlagPrintsVersion(t *testing.T) {
	resetRootCmdFlags(t)
	origVer := Version
	Version = "v1.0.0"
	rootCmd.Version = "v1.0.0"
	defer func() {
		Version = origVer
		rootCmd.Version = origVer
	}()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("--version flag error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "v1.0.0") {
		t.Errorf("output should contain version v1.0.0, got: %q", output)
	}
	if !strings.HasPrefix(output, "potaco v1.0.0") {
		t.Errorf("output should start with 'potaco v1.0.0', got: %q", output)
	}
}

func TestVersionCommandPrintsVersion(t *testing.T) {
	resetRootCmdFlags(t)
	resetVersionCache()
	origVer := Version
	Version = "v1.0.0"
	defer func() { Version = origVer }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Mock the GitHub API to fail so we test graceful degradation.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	origURL := githubReleaseURL
	githubReleaseURL = srv.URL
	defer func() { githubReleaseURL = origURL }()

	rootCmd.SetArgs([]string{"version"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "v1.0.0") {
		t.Errorf("output should contain version v1.0.0, got: %q", output)
	}
}

func TestVersionCommandJSON(t *testing.T) {
	resetRootCmdFlags(t)
	resetVersionCache()
	origVer := Version
	Version = "v1.0.0"
	defer func() { Version = origVer }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"tag_name": "v1.0.0"})
	}))
	defer srv.Close()
	origURL := githubReleaseURL
	githubReleaseURL = srv.URL
	defer func() { githubReleaseURL = origURL }()

	if err := rootCmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatalf("set json flag: %v", err)
	}
	t.Cleanup(func() { _ = rootCmd.PersistentFlags().Set("json", "false") })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version --json error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"current"`) {
		t.Errorf("JSON output should contain 'current' field, got: %q", output)
	}
	if !strings.Contains(output, `"latest"`) {
		t.Errorf("JSON output should contain 'latest' field, got: %q", output)
	}
	if !strings.Contains(output, `"update_available"`) {
		t.Errorf("JSON output should contain 'update_available' field, got: %q", output)
	}
}

func TestVersionCommandUpToDate(t *testing.T) {
	resetRootCmdFlags(t)
	resetVersionCache()
	origVer := Version
	Version = "v1.0.0"
	defer func() { Version = origVer }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"tag_name": "v1.0.0"})
	}))
	defer srv.Close()
	origURL := githubReleaseURL
	githubReleaseURL = srv.URL
	defer func() { githubReleaseURL = origURL }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "up to date") {
		t.Errorf("output should say 'up to date' when versions match, got: %q", output)
	}
}

func TestVersionCommandUpdateAvailable(t *testing.T) {
	resetRootCmdFlags(t)
	resetVersionCache()
	origVer := Version
	Version = "v1.0.0"
	defer func() { Version = origVer }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"tag_name": "v2.0.0"})
	}))
	defer srv.Close()
	origURL := githubReleaseURL
	githubReleaseURL = srv.URL
	defer func() { githubReleaseURL = origURL }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "update available") {
		t.Errorf("output should say 'update available' when latest > current, got: %q", output)
	}
}
