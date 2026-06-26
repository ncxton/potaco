package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
