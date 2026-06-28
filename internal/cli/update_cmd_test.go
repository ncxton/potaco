package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncxton/potaco/internal/config"
)

func resetUpdateCmdFlags(t *testing.T) {
	t.Helper()
	for _, name := range []string{"force"} {
		flag := updateCmd.Flags().Lookup(name)
		if flag == nil {
			return
		}
		_ = flag.Value.Set(flag.DefValue)
		flag.Changed = false
	}
}

func TestUpdateCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "update" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'update' subcommand")
	}
}

func TestUpdateCommandHasUpgradeAlias(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "update" {
			for _, alias := range cmd.Aliases {
				if alias == "upgrade" {
					return
				}
			}
			t.Fatal("update command should have 'upgrade' alias")
		}
	}
}

func TestUpdateAlreadyUpToDate(t *testing.T) {
	resetRootCmdFlags(t)
	resetVersionCache()
	resetUpdateCmdFlags(t)
	t.Setenv("HOME", t.TempDir())
	origVer := Version
	Version = "v1.0.0"
	defer func() { Version = origVer }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	rootCmd.SetArgs([]string{"update"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("update command error: %v", err)
	}
	if !strings.Contains(buf.String(), "up to date") {
		t.Errorf("output should say 'up to date', got: %q", buf.String())
	}
}

func TestUpdateForceBypassesVersionCheck(t *testing.T) {
	resetRootCmdFlags(t)
	resetVersionCache()
	resetUpdateCmdFlags(t)
	t.Setenv("HOME", t.TempDir())
	origVer := Version
	Version = "v1.0.0"
	defer func() { Version = origVer }()

	// Even if versions match, --force should proceed past the version
	// check. We test only that it doesn't print "up to date".
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"tag_name": "v1.0.0"})
	}))
	defer srv.Close()
	origURL := githubReleaseURL
	githubReleaseURL = srv.URL
	defer func() { githubReleaseURL = origURL }()

	installerCalled := false
	installerSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		installerCalled = true
		w.Header().Set("Content-Type", "application/x-sh")
		_, _ = w.Write([]byte("#!/bin/sh\nexit 42\n"))
	}))
	defer installerSrv.Close()
	origInstallScriptURL := installScriptURL
	installScriptURL = func(tag string) string {
		if tag != "v1.0.0" {
			t.Fatalf("tag = %q, want v1.0.0", tag)
		}
		return installerSrv.URL
	}
	defer func() { installScriptURL = origInstallScriptURL }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"update", "--force", "--non-interactive"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when local installer exits non-zero")
	}
	if !installerCalled {
		t.Fatal("expected --force to proceed to installer")
	}
	if strings.Contains(buf.String(), "up to date") {
		t.Errorf("should not say 'up to date' with --force, got: %q", buf.String())
	}
}

func TestInstallUpdateRejectsOversizedInstaller(t *testing.T) {
	resetRootCmdFlags(t)
	resetUpdateCmdFlags(t)
	t.Setenv("HOME", t.TempDir())

	installerSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-sh")
		_, _ = w.Write(bytes.Repeat([]byte("x"), maxInstallScriptBytes+1))
	}))
	defer installerSrv.Close()

	origInstallScriptURL := installScriptURL
	installScriptURL = func(tag string) string { return installerSrv.URL }
	defer func() { installScriptURL = origInstallScriptURL }()

	err := installUpdate(updateCmd, "v1.0.0")
	if err == nil {
		t.Fatal("expected oversized installer error")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error should mention installer size, got: %v", err)
	}
}

func TestUpdateMigratesConfigAfterSuccessfulInstaller(t *testing.T) {
	resetRootCmdFlags(t)
	resetVersionCache()
	resetUpdateCmdFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	origVer := Version
	Version = "v0.9.0"
	defer func() { Version = origVer }()

	releaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"tag_name": "v1.0.0"})
	}))
	defer releaseSrv.Close()
	origURL := githubReleaseURL
	githubReleaseURL = releaseSrv.URL
	defer func() { githubReleaseURL = origURL }()

	installerSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-sh")
		_, _ = w.Write([]byte(`#!/bin/sh
mkdir -p "$HOME/.potaco"
cat > "$HOME/.potaco/config.yaml" <<'YAML'
` + legacyCustomProviderConfigYAML + `
YAML
`))
	}))
	defer installerSrv.Close()
	origInstallScriptURL := installScriptURL
	installScriptURL = func(tag string) string {
		if tag != "v1.0.0" {
			t.Fatalf("tag = %q, want v1.0.0", tag)
		}
		return installerSrv.URL
	}
	defer func() { installScriptURL = origInstallScriptURL }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"update", "--non-interactive"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update command error: %v", err)
	}

	configPath := filepath.Join(tmpHome, ".potaco", "config.yaml")
	loaded, err := config.LoadMultiProvider(configPath)
	if err != nil {
		t.Fatalf("LoadMultiProvider: %v", err)
	}
	if loaded.SchemaVersion != config.CurrentSchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", loaded.SchemaVersion, config.CurrentSchemaVersion)
	}
	if got := loaded.Providers["custom"].Type; got != "openai-compatible" {
		t.Fatalf("custom Type = %q, want openai-compatible", got)
	}
	backups, err := filepath.Glob(configPath + ".bak-*")
	if err != nil {
		t.Fatalf("backup glob: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("backup count = %d, want 1", len(backups))
	}
	if !strings.Contains(buf.String(), "Update complete.") {
		t.Fatalf("output should include update complete, got: %q", buf.String())
	}
}
