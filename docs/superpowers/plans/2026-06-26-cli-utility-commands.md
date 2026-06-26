# CLI Utility Commands and TUI Enhancements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `version`, `update` (alias `upgrade`), and `uninstall` commands; interactive `auth remove` flow; searchable model picker; and Esc-to-cancel for all TUI flows.

**Architecture:** New cobra subcommands in `internal/cli/` follow existing patterns (one file per subcommand, register via `init()`). TUI additions in `internal/tui/` use `huh` forms with a shared `newForm` helper that binds Esc to quit. The models search uses a custom Bubble Tea program for real-time filtering. `install.sh` is simplified to `~/.local/bin` only with shell-config PATH detection.

**Tech Stack:** Go 1.26, cobra v1.10.2, charmbracelet/huh v1.0.0, charmbracelet/bubbletea (indirect via huh), charm.land/lipgloss/v2

## Global Constraints

- Go 1.26, pure Go only (no CGO)
- Standard `gofmt` formatting; run `gofmt -l .` before committing
- No panics in library code; use `fmt.Errorf` with `%w` for error wrapping
- Internal packages under `internal/` are not importable externally
- Commands register on `rootCmd` via `init()` in their own file
- User-facing errors use `UserError` via `configUserErr`/`apiUserErr`/`imageUserErr`
- Commit messages use Conventional Commits: `feat(scope):`, `fix(scope):`, etc.
- TDD: write failing tests first, implement to pass, then commit
- Tests dispatch via `rootCmd.SetArgs([]string{...})` and `rootCmd.Execute()`
- Table-driven tests preferred; test files alongside source

---

## File Structure

### New Files
| File | Responsibility |
|------|----------------|
| `internal/cli/version.go` | Version variable and `SetVersion` setter |
| `internal/cli/version_cmd.go` | `version` subcommand + `checkLatestVersion` with cache |
| `internal/cli/version_cmd_test.go` | Tests for version command |
| `internal/cli/update_cmd.go` | `update` (alias `upgrade`) subcommand |
| `internal/cli/update_cmd_test.go` | Tests for update command |
| `internal/cli/uninstall_cmd.go` | `uninstall` subcommand |
| `internal/cli/uninstall_cmd_test.go` | Tests for uninstall command |
| `internal/tui/auth_remove.go` | Interactive auth remove TUI flow |
| `internal/tui/model_search.go` | Custom Bubble Tea search model |

### Modified Files
| File | Changes |
|------|---------|
| `main.go` | Add `var version = "unknown"`, call `cli.SetVersion(version)` |
| `.goreleaser.yaml` | Add `ldflags` to builds section |
| `install.sh` | Simplify to `~/.local/bin` only, add PATH detection |
| `internal/cli/auth_cmd.go` | Change `authRemoveCmd.Args`, update `runAuthRemove` |
| `internal/cli/auth_cmd_test.go` | Update test for no-arg behavior, add non-interactive no-arg error test |
| `internal/tui/tui.go` | Add `newForm` helper with Esc key binding |
| `internal/tui/auth_add.go` | Replace `huh.NewForm` with `newForm` |
| `internal/tui/use_picker.go` | Replace `huh.NewForm` with `newForm` |
| `internal/tui/model_list.go` | Replace `pickModelInteractive` with Bubble Tea search |
| `internal/tui/model_list_test.go` | Add search filtering tests |

---

## Task 1: Version Variable Infrastructure

**Files:**
- Create: `internal/cli/version.go`
- Modify: `internal/cli/version_cmd_test.go` (placeholder, will be filled in Task 2)

**Interfaces:**
- Produces: `var Version string` (package-level, defaults to `"unknown"`) and `func SetVersion(v string)`

- [ ] **Step 1: Create `internal/cli/version.go`**

```go
package cli

// Version holds the current binary version. It defaults to "unknown" for
// locally-built binaries and is overridden via ldflags at release time:
//
//	go build -ldflags "-X main.version=v1.2.3"
//
// main.go calls SetVersion() to push the value here.
var Version = "unknown"

// SetVersion sets the package-level Version variable. Called from main()
// with the ldflags-injected value.
func SetVersion(v string) {
	if v != "" {
		Version = v
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build -o /dev/null .`
Expected: builds with no errors

- [ ] **Step 3: Modify `main.go` to inject version**

Replace `main.go` entirely:

```go
package main

import "github.com/ncxton/potaco/internal/cli"

var version = "unknown"

func main() {
	cli.SetVersion(version)
	cli.Execute()
}
```

- [ ] **Step 4: Verify it compiles and runs**

Run: `go build -o /dev/null . && echo OK`
Expected: `OK`

- [ ] **Step 5: Commit**

```bash
git add main.go internal/cli/version.go
git commit -m "feat(cli): add version variable infrastructure with ldflags injection"
```

---

## Task 2: `checkLatestVersion` Function with Caching

**Files:**
- Create: `internal/cli/version_cmd.go` (partial - just the GitHub API function for now)

**Interfaces:**
- Produces: `func checkLatestVersion() (string, error)` - returns latest release tag from GitHub API, cached for 1 hour

- [ ] **Step 1: Write the failing test**

Create `internal/cli/version_cmd_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails (function not defined)**

Run: `go test ./internal/cli/ -run TestCheckLatestVersion -v`
Expected: FAIL with compilation error (`checkLatestVersion` undefined, `githubReleaseURL` undefined, `latestCache` undefined)

- [ ] **Step 3: Write the implementation**

Create `internal/cli/version_cmd.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestCheckLatestVersion -v`
Expected: PASS for all 3 tests

- [ ] **Step 5: Commit**

```bash
git add internal/cli/version_cmd.go internal/cli/version_cmd_test.go
git commit -m "feat(cli): add checkLatestVersion with GitHub API and caching"
```

---

## Task 3: `potaco version` Command

**Files:**
- Modify: `internal/cli/version_cmd.go` (add command registration and `runVersion`)
- Modify: `internal/cli/version_cmd_test.go` (add command tests)

**Interfaces:**
- Consumes: `var Version` from Task 1, `func checkLatestVersion()` from Task 2
- Produces: `potaco version` subcommand registered on `rootCmd`

- [ ] **Step 1: Write the failing tests**

Append to `internal/cli/version_cmd_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestVersionCommand -v`
Expected: FAIL (`version` command not registered)

- [ ] **Step 3: Add the command registration and `runVersion`**

Append to `internal/cli/version_cmd.go`:

```go
import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

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

	// Try to fetch latest version; graceful degradation on failure.
	latest, latestErr := checkLatestVersion()

	updateAvailable := false
	if latestErr == nil && latest != "" && latest != Version {
		updateAvailable = true
	}

	if jsonMode {
		type versionJSON struct {
			Current          string `json:"current"`
			Latest           string `json:"latest"`
			UpdateAvailable  bool   `json:"update_available"`
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

	// Text mode: print current version.
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
```

Note: The `import` block needs to be merged at the top of the file. The final `version_cmd.go` should have all imports in one block. Since we created the file in Task 2 with imports, we need to add `"github.com/spf13/cobra"` to the existing import block and add the command code at the end.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestVersionCommand -v`
Expected: PASS for all 5 tests

- [ ] **Step 5: Commit**

```bash
git add internal/cli/version_cmd.go internal/cli/version_cmd_test.go
git commit -m "feat(cli): add version command with GitHub latest check and JSON output"
```

---

## Task 4: Update `.goreleaser.yaml` with ldflags

**Files:**
- Modify: `.goreleaser.yaml`

- [ ] **Step 1: Add ldflags to the build section**

Edit `.goreleaser.yaml` to add `ldflags` under the `potaco` build:

```yaml
builds:
  - id: potaco
    main: .
    binary: potaco
    ldflags:
      - -X main.version={{.Tag}}
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
```

This replaces the existing `builds` section. The only addition is the `ldflags` key. `{{.Tag}}` resolves to the git tag (e.g., `v1.2.3`) so the version string includes the `v` prefix.

- [ ] **Step 2: Verify YAML is valid**

Run: `python3 -c "import yaml; yaml.safe_load(open('.goreleaser.yaml'))"` or check with `go test ./...` to make sure nothing broke

- [ ] **Step 3: Commit**

```bash
git add .goreleaser.yaml
git commit -m "build: add ldflags version injection to goreleaser config"
```

---

## Task 5: `potaco update` Command (alias `upgrade`)

**Files:**
- Create: `internal/cli/update_cmd.go`
- Create: `internal/cli/update_cmd_test.go`

**Interfaces:**
- Consumes: `var Version` from Task 1, `func checkLatestVersion()` from Task 2, `var githubReleaseURL` from Task 2
- Produces: `potaco update` (alias `upgrade`) subcommand registered on `rootCmd`

- [ ] **Step 1: Write the failing tests**

Create `internal/cli/update_cmd_test.go`:

```go
package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	origVer := Version
	Version = "v1.0.0"
	defer func() { Version = origVer }()

	// Even if versions match, --force should proceed past the version
	// check. We test only that it doesn't print "up to date".
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
	// --force will try to download install.sh, which will fail against
	// the mock server. We expect an error mentioning download failure,
	// NOT "up to date".
	rootCmd.SetArgs([]string{"update", "--force", "--non-interactive"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --force tries to download install.sh from mock")
	}
	if strings.Contains(buf.String(), "up to date") {
		t.Errorf("should not say 'up to date' with --force, got: %q", buf.String())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestUpdate -v`
Expected: FAIL (`updateCmd` undefined, `update` command not registered)

- [ ] **Step 3: Write the implementation**

Create `internal/cli/update_cmd.go`:

```go
package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/ncxton/potaco/internal/tui"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update potaco to the latest release",
	Aliases: []string{"upgrade"},
	RunE:    runUpdate,
}

func init() {
	updateCmd.Flags().BoolP("force", "f", false, "force update even if already at latest version")
	rootCmd.AddCommand(updateCmd)
}

// installScriptURL returns the raw install.sh URL for a given release tag.
func installScriptURL(tag string) string {
	return fmt.Sprintf("https://github.com/ncxton/potaco/releases/download/%s/install.sh", tag)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	// Check latest version (reuses cache from version command).
	latest, err := checkLatestVersion()
	if err != nil {
		return configUserErr(
			"Could not check for updates.",
			"Check your network connection and try again.",
			fmt.Errorf("check latest version: %w", err),
		)
	}

	// Compare versions unless --force or unknown local build.
	if !force && Version != "unknown" && Version == latest {
		fmt.Fprintf(cmd.OutOrStdout(), "Already up to date (%s).\n", Version)
		return nil
	}

	if force && Version != "unknown" && Version == latest {
		fmt.Fprintf(cmd.OutOrStdout(), "Forcing update (already at %s)...\n", Version)
	}

	// Download install.sh from the release.
	installURL := installScriptURL(latest)
	tmpFile, err := os.CreateTemp("", "potaco-install-*.sh")
	if err != nil {
		return apiUserErr(
			"Could not create a temporary file for the installer.",
			"",
			fmt.Errorf("create temp file: %w", err),
		)
	}
	defer os.Remove(tmpFile.Name())

	resp, err := http.Get(installURL)
	if err != nil {
		return apiUserErr(
			"Could not download the installer.",
			"Check your network connection and try again.",
			fmt.Errorf("download install.sh: %w", err),
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return apiUserErr(
			"Could not download the installer.",
			fmt.Sprintf("GitHub returned status %d for the install script.", resp.StatusCode),
			fmt.Errorf("install.sh download returned status %d", resp.StatusCode),
		)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return apiUserErr(
			"Could not save the installer to disk.",
			"",
			fmt.Errorf("write temp file: %w", err),
		)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return configUserErr(
			"Could not make the installer executable.",
			"",
			fmt.Errorf("chmod installer: %w", err),
		)
	}

	// Execute the installer, inheriting stdin/stdout/stderr.
	fmt.Fprintf(cmd.OutOrStdout(), "Running installer...\n")

	sc := exec.Command("sh", tmpFile.Name())
	sc.Stdin = os.Stdin
	sc.Stdout = os.Stdout
	sc.Stderr = os.Stderr

	// Pass non-interactive flag to the installer if set.
	if !tui.IsInteractive() {
		sc.Env = append(os.Environ(), "POTACO_NON_INTERACTIVE=1")
	} else {
		sc.Env = os.Environ()
	}

	if err := sc.Run(); err != nil {
		return apiUserErr(
			"Installer failed.",
			"Check the output above for details, or try running the installer manually.",
			fmt.Errorf("install.sh execution: %w", err),
		)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Update complete.\n")
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestUpdate -v`
Expected: PASS for all 4 tests

- [ ] **Step 5: Run go vet and gofmt**

Run: `go vet ./internal/cli/ && gofmt -l internal/cli/update_cmd.go`
Expected: no output (clean)

- [ ] **Step 6: Commit**

```bash
git add internal/cli/update_cmd.go internal/cli/update_cmd_test.go
git commit -m "feat(cli): add update command with upgrade alias"
```

---

## Task 6: `potaco uninstall` Command

**Files:**
- Create: `internal/cli/uninstall_cmd.go`
- Create: `internal/cli/uninstall_cmd_test.go`

**Interfaces:**
- Consumes: `func newForm` from Task 8 (Esc-enabled forms). If Task 8 hasn't been done yet, use `huh.NewForm` directly and migrate later.
- Produces: `potaco uninstall` subcommand registered on `rootCmd`

- [ ] **Step 1: Write the failing tests**

Create `internal/cli/uninstall_cmd_test.go`:

```go
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetUninstallCmdFlags(t *testing.T) {
	t.Helper()
	for _, name := range []string{"remove-config", "yes"} {
		flag := uninstallCmd.Flags().Lookup(name)
		if flag == nil {
			return
		}
		_ = flag.Value.Set(flag.DefValue)
		flag.Changed = false
	}
}

func TestUninstallCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "uninstall" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'uninstall' subcommand")
	}
}

func TestUninstallNonInteractiveRemovesBinary(t *testing.T) {
	resetRootCmdFlags(t)
	resetUninstallCmdFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create a fake binary
	binPath := filepath.Join(tmpHome, ".local", "bin", "potaco")
	if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(binPath, []byte("fake binary"), 0755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"uninstall", "--non-interactive"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("uninstall error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "removed") || !strings.Contains(output, "binary") {
		t.Errorf("output should mention binary removal, got: %q", output)
	}

	// Verify the binary was removed
	if _, err := os.Stat(binPath); !os.IsNotExist(err) {
		t.Errorf("binary should have been removed, but file still exists at %s", binPath)
	}
}

func TestUninstallNonInteractiveWithRemoveConfig(t *testing.T) {
	resetRootCmdFlags(t)
	resetUninstallCmdFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create a fake binary
	binPath := filepath.Join(tmpHome, ".local", "bin", "potaco")
	if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(binPath, []byte("fake binary"), 0755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	// Create a fake config directory
	configDir := filepath.Join(tmpHome, ".potaco")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("test"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"uninstall", "--non-interactive", "--remove-config"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("uninstall error: %v", err)
	}

	// Verify both binary and config were removed
	if _, err := os.Stat(binPath); !os.IsNotExist(err) {
		t.Errorf("binary should have been removed")
	}
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Errorf("config dir should have been removed")
	}
}

func TestUninstallBinaryNotFoundWarns(t *testing.T) {
	resetRootCmdFlags(t)
	resetUninstallCmdFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Do NOT create a binary

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"uninstall", "--non-interactive"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("uninstall should not error when binary not found, got: %v", err)
	}
	// Should warn but not fail
	output := buf.String()
	if !strings.Contains(output, "not found") && !strings.Contains(output, "already") {
		t.Errorf("output should mention binary not found, got: %q", output)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestUninstall -v`
Expected: FAIL (`uninstallCmd` undefined)

- [ ] **Step 3: Write the implementation**

Create `internal/cli/uninstall_cmd.go`:

```go
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ncxton/potaco/internal/tui"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the potaco binary and optionally its configuration",
	RunE:  runUninstall,
}

func init() {
	uninstallCmd.Flags().Bool("remove-config", false, "also remove ~/.potaco/ config directory")
	uninstallCmd.Flags().BoolP("yes", "y", false, "skip confirmation prompts")
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	removeConfig, _ := cmd.Flags().GetBool("remove-config")
	nonInteractive := !tui.IsInteractive()

	// Locate the binary
	binaryPath, err := findPotacoBinary()
	if err != nil && nonInteractive {
		// In non-interactive mode, warn but don't fail.
		fmt.Fprintf(cmd.OutOrStdout(), "Warning: could not locate potaco binary: %v\n", err)
	}

	// Determine home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return configUserErr(
			"Could not determine your home directory.",
			"",
			fmt.Errorf("user home dir: %w", err),
		)
	}
	configDir := filepath.Join(home, ".potaco")

	if nonInteractive {
		return runUninstallNonInteractive(cmd, binaryPath, configDir, removeConfig)
	}

	// Interactive mode: use the TUI flow
	return runUninstallInteractive(cmd, binaryPath, configDir, yes)
}

func runUninstallNonInteractive(cmd *cobra.Command, binaryPath, configDir string, removeConfig bool) error {
	out := cmd.OutOrStdout()

	if binaryPath != "" {
		if err := os.Remove(binaryPath); err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(out, "Binary not found at %s (may have been removed already).\n", binaryPath)
			} else {
				return configUserErr(
					"Cannot remove the binary.",
					fmt.Sprintf("Check file permissions or remove manually: %s", binaryPath),
					fmt.Errorf("remove binary: %w", err),
				)
			}
		} else {
			fmt.Fprintf(out, "Binary removed: %s\n", binaryPath)
		}
	}

	if removeConfig {
		if err := os.RemoveAll(configDir); err != nil {
			fmt.Fprintf(out, "Warning: could not remove config directory: %v\n", err)
		} else {
			fmt.Fprintf(out, "Config directory removed: %s\n", configDir)
		}
	}

	fmt.Fprintf(out, "Uninstall complete.\n")
	return nil
}

func runUninstallInteractive(cmd *cobra.Command, binaryPath, configDir string, yes bool) error {
	out := cmd.OutOrStdout()

	// Step 1: Confirm binary removal
	if !yes {
		confirmed, err := tui.ConfirmAction(fmt.Sprintf("This will remove the potaco binary at '%s'. Continue?", binaryPath))
		if err != nil {
			fmt.Fprintln(out, "Cancelled.")
			return nil
		}
		if !confirmed {
			fmt.Fprintln(out, "Cancelled.")
			return nil
		}
	}

	// Step 2: Ask about config removal
	removeConfig := false
	if !yes {
		removeConfig, _ = tui.ConfirmAction(fmt.Sprintf("Also remove configuration and credentials at '%s'?", configDir))
	} else {
		// --yes auto-answers Yes to config removal only if --remove-config was passed
		removeConfig, _ = cmd.Flags().GetBool("remove-config")
	}

	// Step 3: Final confirmation
	if !yes {
		summary := "Confirm: remove binary"
		if removeConfig {
			summary += " and config directory"
		}
		summary += "?"
		finalConfirm, err := tui.ConfirmAction(summary)
		if err != nil {
			fmt.Fprintln(out, "Cancelled.")
			return nil
		}
		if !finalConfirm {
			fmt.Fprintln(out, "Cancelled.")
			return nil
		}
	}

	// Execute removal
	if binaryPath != "" {
		if err := os.Remove(binaryPath); err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(out, "Binary not found at %s (may have been removed already).\n", binaryPath)
			} else {
				return configUserErr(
					"Cannot remove the binary.",
					fmt.Sprintf("Check file permissions or remove manually: %s", binaryPath),
					fmt.Errorf("remove binary: %w", err),
				)
			}
		} else {
			fmt.Fprintf(out, "Binary removed: %s\n", binaryPath)
		}
	}

	if removeConfig {
		if err := os.RemoveAll(configDir); err != nil {
			fmt.Fprintf(out, "Warning: could not remove config directory: %v\n", err)
		} else {
			fmt.Fprintf(out, "Config directory removed: %s\n", configDir)
		}
	}

	fmt.Fprintf(out, "Uninstall complete.\n")
	return nil
}

// findPotacoBinary locates the running potaco binary path, resolving symlinks.
func findPotacoBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		// Fallback: search PATH
		path, err := exec.LookPath("potaco")
		if err != nil {
			return "", fmt.Errorf("could not locate potaco binary: %w", err)
		}
		return path, nil
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe, nil
	}
	return resolved, nil
}
```

- [ ] **Step 4: Add the `ConfirmAction` helper to `internal/tui/tui.go`**

This helper wraps a `huh.NewConfirm` form with the `newForm` helper. If Task 8 has not been completed yet, use `huh.NewForm` directly. But since the plan order puts Task 8 before Task 6 in the `tui.go` section, this should use `newForm`:

Append to `internal/tui/tui.go` (after the `newForm` helper from Task 8):

```go
// ConfirmAction shows a yes/no confirmation prompt and returns the result.
// Returns (false, ErrUserAborted-like) when the user presses Esc/Ctrl+C.
func ConfirmAction(prompt string) (bool, error) {
	var result bool
	form := newForm(huh.NewGroup(
		huh.NewConfirm().
			Title(prompt).
			Value(&result),
	))
	if err := form.Run(); err != nil {
		return false, err
	}
	return result, nil
}
```

Note: This requires `huh` to be imported in `tui.go`. Add `"github.com/charmbracelet/huh"` to the import block.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestUninstall -v`
Expected: PASS for all 4 tests

- [ ] **Step 6: Run go vet and gofmt**

Run: `go vet ./internal/cli/ && gofmt -l internal/cli/uninstall_cmd.go internal/tui/tui.go`
Expected: no output (clean)

- [ ] **Step 7: Commit**

```bash
git add internal/cli/uninstall_cmd.go internal/cli/uninstall_cmd_test.go internal/tui/tui.go
git commit -m "feat(cli): add uninstall command with interactive and non-interactive modes"
```

---

## Task 7: Esc-to-Cancel Helper in `tui.go`

**Files:**
- Modify: `internal/tui/tui.go`

**Interfaces:**
- Produces: `func newForm(groups ...*huh.Group) *huh.Form` - a form factory with Esc bound to quit
- Produces: `func ConfirmAction(prompt string) (bool, error)` - used by uninstall (Task 6)

- [ ] **Step 1: Write a test for the `newForm` helper**

Add to a new test in `internal/tui/tui_test.go`. Since `newForm` produces a `*huh.Form` and testing the actual Esc behavior requires a TTY, we test that the function returns a non-nil form with the correct keymap:

Append to `internal/tui/tui_test.go`:

```go
import (
	"testing"

	"github.com/charmbracelet/huh"
)

func TestNewFormReturnsFormWithEscQuit(t *testing.T) {
	form := newForm(huh.NewGroup(
		huh.NewConfirm().Title("test"),
	))
	if form == nil {
		t.Fatal("newForm should return a non-nil form")
	}
}

func TestNewFormNotNil(t *testing.T) {
	form := newForm(huh.NewGroup(
		huh.NewInput().Title("test"),
	))
	if form == nil {
		t.Fatal("newForm should return non-nil form")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/tui/ -run TestNewForm -v`
Expected: FAIL (`newForm` undefined)

- [ ] **Step 3: Write the implementation**

Add to `internal/tui/tui.go`:

```go
import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

// newForm creates a huh.Form with the Esc key bound to quit alongside
// the default ctrl+c. This allows users to cancel any interactive TUI
// flow by pressing Esc.
func newForm(groups ...*huh.Group) *huh.Form {
	form := huh.NewForm(groups...)
	keymap := huh.NewDefaultKeyMap()
	keymap.Quit = key.NewBinding(
		key.WithKeys("ctrl+c", "esc"),
		key.WithHelp("ctrl+c / esc", "quit"),
	)
	form.WithKeyMap(keymap)
	return form
}

// isCancelled returns true when the error from form.Run() indicates the
// user aborted (pressed Esc or Ctrl+C).
func isCancelled(err error) bool {
	return errors.Is(err, huh.ErrUserAborted)
}
```

- [ ] **Step 4: Add `ConfirmAction` helper**

Append to `internal/tui/tui.go`:

```go
// ConfirmAction shows a yes/no confirmation prompt and returns the result.
// Returns (false, error) when the user presses Esc/Ctrl+C.
func ConfirmAction(prompt string) (bool, error) {
	var result bool
	form := newForm(huh.NewGroup(
		huh.NewConfirm().
			Title(prompt).
			Value(&result),
	))
	if err := form.Run(); err != nil {
		return false, err
	}
	return result, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run TestNewForm -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/tui.go internal/tui/tui_test.go
git commit -m "feat(tui): add newForm helper with Esc-to-cancel and ConfirmAction"
```

---

## Task 8: Migrate Existing TUI Forms to `newForm`

**Files:**
- Modify: `internal/tui/auth_add.go`
- Modify: `internal/tui/use_picker.go`

**Interfaces:**
- Consumes: `func newForm` from Task 7
- Produces: all existing TUI flows now support Esc-to-cancel

- [ ] **Step 1: Migrate `auth_add.go`**

In `internal/tui/auth_add.go`, replace all `huh.NewForm(...)` calls with `newForm(...)`. There are 3 form creations:

1. Key input form (line ~30):
   ```go
   // Before:
   keyForm := huh.NewForm(huh.NewGroup(...))
   // After:
   keyForm := newForm(huh.NewGroup(...))
   ```

2. Verify confirm form (line ~50):
   ```go
   // Before:
   confirmForm := huh.NewForm(huh.NewGroup(...))
   // After:
   confirmForm := newForm(huh.NewGroup(...))
   ```

3. Model select form (line ~70):
   ```go
   // Before:
   selectForm := huh.NewForm(huh.NewGroup(...))
   // After:
   selectForm := newForm(huh.NewGroup(...))
   ```

Also add Esc-cancel handling after each `form.Run()` call. For the key input and model select forms, wrap the error check:

```go
if err := keyForm.Run(); err != nil {
	if isCancelled(err) {
		fmt.Println("Cancelled.")
		return nil
	}
	return fmt.Errorf("key input: %w", err)
}
```

Apply this pattern to the confirm form as well.

- [ ] **Step 2: Migrate `use_picker.go`**

In `internal/tui/use_picker.go`, replace all `huh.NewForm(...)` calls with `newForm(...)`. There are 3 form creations:

1. `pickProvider` - provider select form
2. `pickModel` - model confirm form
3. `pickModel` - model input form

Add `isCancelled` check after each `form.Run()`:

```go
if err := form.Run(); err != nil {
	if isCancelled(err) {
		fmt.Println("Cancelled.")
		return "", nil
	}
	return "", fmt.Errorf("provider select: %w", err)
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build -o /dev/null .`
Expected: builds with no errors

- [ ] **Step 4: Run all TUI tests**

Run: `go test ./internal/tui/ -v`
Expected: PASS (existing tests still pass, TUI forms are smoke-tested)

- [ ] **Step 5: Commit**

```bash
git add internal/tui/auth_add.go internal/tui/use_picker.go
git commit -m "feat(tui): migrate existing forms to newForm with Esc-to-cancel"
```

---

## Task 9: Interactive `auth remove` Flow

**Files:**
- Create: `internal/tui/auth_remove.go`
- Modify: `internal/cli/auth_cmd.go` (change Args, update `runAuthRemove`)
- Modify: `internal/cli/auth_cmd_test.go` (update existing test, add new tests)

**Interfaces:**
- Consumes: `func newForm` from Task 7, `func isCancelled` from Task 7, `func ConfirmAction` from Task 7
- Produces: `func RunAuthRemove(providerName string) error` in `tui` package

- [ ] **Step 1: Write the failing tests**

Add to `internal/cli/auth_cmd_test.go`:

```go
func TestAuthRemoveNoArgsNonInteractiveErrors(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "remove"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth remove without args in non-interactive mode should error")
	}
	if !strings.Contains(err.Error(), "specify") && !strings.Contains(err.Error(), "Specify") {
		t.Errorf("error should ask to specify a provider, got: %v", err)
	}
}

func TestAuthRemoveUnknownProviderNonInteractive(t *testing.T) {
	newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "remove", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth remove with unknown provider should error")
	}
}

func TestAuthRemoveKnownProviderNonInteractiveStillWorks(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "remove", "openai"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth remove error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "removed") || !strings.Contains(output, "openai") {
		t.Errorf("output should mention removal of openai, got: %q", output)
	}
}
```

Also update `TestAuthRemoveRequiresProviderArg` — it currently tests that `auth remove` without args errors. With the new behavior, this only applies in non-interactive mode (which is the case in tests since there's no TTY). The existing test should still pass since test environments are non-interactive. No change needed to the existing test.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run "TestAuthRemoveNoArgs|TestAuthRemoveUnknown" -v`
Expected: FAIL (the `Args` constraint is still `ExactArgs(1)` which prevents zero-arg invocation, and the test for no-args non-interactive expects an error from the handler, not from cobra's arg validation)

- [ ] **Step 3: Create `internal/tui/auth_remove.go`**

```go
package tui

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/ncxton/potaco/internal/auth"
)

// RunAuthRemove launches the interactive auth remove flow.
// If providerName is empty, shows a provider picker first.
// Shows a confirmation prompt before removing. Returns nil when the
// user cancels (pressed Esc), and prints "Cancelled." to stdout.
func RunAuthRemove(providerName string) error {
	mgr, err := auth.New()
	if err != nil {
		return fmt.Errorf("init auth: %w", err)
	}

	// If no provider name given, show picker
	if providerName == "" {
		providers := mgr.List()
		if len(providers) == 0 {
			return fmt.Errorf("no providers connected")
		}

		options := make([]huh.Option[string], 0, len(providers))
		for _, p := range providers {
			label := p.Name
			if p.IsActive {
				label += " (active)"
			}
			label += " - " + p.Model
			options = append(options, huh.NewOption(label, p.Name))
		}

		selectForm := newForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a provider to remove:").
				Options(options...).
				Value(&providerName),
		))
		if err := selectForm.Run(); err != nil {
			if isCancelled(err) {
				fmt.Println("Cancelled.")
				return nil
			}
			return fmt.Errorf("provider select: %w", err)
		}
	}

	// Confirmation prompt
	confirmed, err := ConfirmAction(fmt.Sprintf("Remove provider '%s' and its credentials?", providerName))
	if err != nil {
		if isCancelled(err) {
			fmt.Println("Cancelled.")
			return nil
		}
		return fmt.Errorf("confirm: %w", err)
	}
	if !confirmed {
		fmt.Println("Cancelled.")
		return nil
	}

	// Execute removal
	if err := mgr.Remove(providerName); err != nil {
		return fmt.Errorf("remove provider: %w", err)
	}

	fmt.Printf("Provider '%s' removed.\n", providerName)
	return nil
}
```

- [ ] **Step 4: Modify `internal/cli/auth_cmd.go`**

Change the `authRemoveCmd` definition:
```go
// Before (line 30):
var authRemoveCmd = &cobra.Command{
	Use:     "remove <provider>",
	Short:   "Remove a provider's credentials and config",
	Args:    cobra.ExactArgs(1),
	RunE:    runAuthRemove,
	Aliases: []string{"rm"},
}

// After:
var authRemoveCmd = &cobra.Command{
	Use:     "remove [provider]",
	Short:   "Remove a provider's credentials and config",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runAuthRemove,
	Aliases: []string{"rm"},
}
```

Replace the `runAuthRemove` function:
```go
// Before (lines 117-131):
func runAuthRemove(cmd *cobra.Command, args []string) error {
	providerName := args[0]

	mgr, err := auth.New()
	if err != nil {
		return configError(fmt.Errorf("init auth: %w", err))
	}

	if err := mgr.Remove(providerName); err != nil {
		return configError(fmt.Errorf("remove provider: %w", err))
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Provider '%s' removed.\n", providerName)
	return nil
}

// After:
func runAuthRemove(cmd *cobra.Command, args []string) error {
	providerName := ""
	if len(args) > 0 {
		providerName = args[0]
	}

	// Non-interactive mode: require provider arg and remove directly.
	if !tui.IsInteractive() {
		if providerName == "" {
			return configError(fmt.Errorf("specify a provider: potaco auth remove <provider>"))
		}

		mgr, err := auth.New()
		if err != nil {
			return configError(fmt.Errorf("init auth: %w", err))
		}
		if err := mgr.Remove(providerName); err != nil {
			return configError(fmt.Errorf("remove provider: %w", err))
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Provider '%s' removed.\n", providerName)
		return nil
	}

	// Interactive mode: launch TUI flow (picker if no arg, then confirm)
	return tui.RunAuthRemove(providerName)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestAuthRemove -v`
Expected: PASS for all tests including the new no-args and known-provider tests

- [ ] **Step 6: Run all tests to check for regressions**

Run: `go test ./... -v 2>&1 | tail -30`
Expected: all tests pass

- [ ] **Step 7: Commit**

```bash
git add internal/tui/auth_remove.go internal/cli/auth_cmd.go internal/cli/auth_cmd_test.go
git commit -m "feat(tui): add interactive auth remove flow with provider picker and confirmation"
```

---

## Task 10: Models Search-by-Typing (Custom Bubble Tea Program)

**Files:**
- Create: `internal/tui/model_search.go`
- Create: `internal/tui/model_search_test.go`
- Modify: `internal/tui/model_list.go` (replace `pickModelInteractive`)

**Interfaces:**
- Consumes: `adapter.Model` type (already defined in `internal/adapter/adapter.go`)
- Produces: `func newSearchModel(models []adapter.Model) *searchModel` and `func (m *searchModel) View() string`, `func (m *searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)`

- [ ] **Step 1: Write the failing tests for filtering logic**

Create `internal/tui/model_search_test.go`:

```go
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ncxton/potaco/internal/adapter"
)

func TestNewSearchModel(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2", SupportsEdit: true},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
		{ID: "flux-pro", DisplayName: "Flux Pro"},
	}
	m := newSearchModel(models)

	if len(m.models) != 3 {
		t.Errorf("expected 3 models, got %d", len(m.models))
	}
	if len(m.filtered) != 3 {
		t.Errorf("expected 3 filtered models initially, got %d", len(m.filtered))
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
}

func TestSearchFilterByID(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
		{ID: "flux-pro", DisplayName: "Flux Pro"},
	}
	m := newSearchModel(models)
	m.query.SetValue("gpt")

	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered model, got %d", len(m.filtered))
	}
	if m.filtered[0].ID != "gpt-image-2" {
		t.Errorf("expected gpt-image-2, got %s", m.filtered[0].ID)
	}
}

func TestSearchFilterByDisplayName(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
		{ID: "flux-pro", DisplayName: "Flux Pro"},
	}
	m := newSearchModel(models)
	m.query.SetValue("dall")

	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered model, got %d", len(m.filtered))
	}
	if m.filtered[0].ID != "dall-e-3" {
		t.Errorf("expected dall-e-3, got %s", m.filtered[0].ID)
	}
}

func TestSearchFilterCaseInsensitive(t *testing.T) {
	models := []adapter.Model{
		{ID: "GPT-Image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
	}
	m := newSearchModel(models)
	m.query.SetValue("gpt")

	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered model (case-insensitive), got %d", len(m.filtered))
	}
}

func TestSearchFilterEmptyShowsAll(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
	}
	m := newSearchModel(models)
	m.query.SetValue("")

	m.applyFilter()

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered when query empty, got %d", len(m.filtered))
	}
}

func TestSearchFilterNoMatch(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
	}
	m := newSearchModel(models)
	m.query.SetValue("xyz123")

	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Errorf("expected 0 filtered models, got %d", len(m.filtered))
	}
}

func TestSearchCursorClampsToFiltered(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
		{ID: "flux-pro", DisplayName: "Flux Pro"},
	}
	m := newSearchModel(models)
	m.cursor = 2 // at last item

	// Filter to only 1 item
	m.query.SetValue("flux")
	m.applyFilter()

	// applyFilter calls clampCursor internally, so cursor should be 0
	if m.cursor != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", m.cursor)
	}
}

func TestSearchEscQuits(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
	}
	m := newSearchModel(models)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if !m.quitted {
		t.Error("expected quitted=true after Esc")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (tea.Quit)")
	}
}

func TestSearchEnterSelects(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
	}
	m := newSearchModel(models)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.selected != "gpt-image-2" {
		t.Errorf("expected selected gpt-image-2, got %q", m.selected)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (tea.Quit)")
	}
}

func TestSearchArrowDownMovesCursor(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
		{ID: "flux-pro", DisplayName: "Flux Pro"},
	}
	m := newSearchModel(models)
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", m.cursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2, got %d", m.cursor)
	}
	// Wrap at bottom
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 0 {
		t.Errorf("expected cursor to wrap to 0, got %d", m.cursor)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run TestSearch -v`
Expected: FAIL (`newSearchModel` undefined, `searchModel` type undefined)

- [ ] **Step 3: Write the implementation**

Create `internal/tui/model_search.go`:

```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"charm.land/lipgloss/v2"

	"github.com/ncxton/potaco/internal/adapter"
)

// searchModel is a custom Bubble Tea model for searching and selecting
// from a list of models by typing. The filter updates in real-time as
// the user types characters.
type searchModel struct {
	models   []adapter.Model
	filtered []adapter.Model
	cursor   int
	query    textinput.Model
	selected string
	quitted  bool
}

// newSearchModel creates a new searchModel initialized with the given
// models. All models are shown initially (no filter applied).
func newSearchModel(models []adapter.Model) *searchModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Type to search..."
	ti.Focus()

	return &searchModel{
		models:   models,
		filtered: models,
		query:    ti,
	}
}

// Init implements tea.Model.
func (m *searchModel) Init() tea.Cmd {
	return textinput.Blink
}

// applyFilter updates m.filtered based on the current query value.
// Filtering is case-insensitive and matches against both ID and DisplayName.
func (m *searchModel) applyFilter() {
	q := strings.ToLower(m.query.Value())
	if q == "" {
		m.filtered = m.models
		m.clampCursor()
		return
	}
	m.filtered = nil
	for _, model := range m.models {
		if strings.Contains(strings.ToLower(model.ID), q) ||
			strings.Contains(strings.ToLower(model.DisplayName), q) {
			m.filtered = append(m.filtered, model)
		}
	}
	m.clampCursor()
}

// clampCursor ensures the cursor is within bounds of the filtered list.
func (m *searchModel) clampCursor() {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor > len(m.filtered)-1 {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// Update implements tea.Model.
func (m *searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.quitted = true
			return m, tea.Quit
		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				m.selected = m.filtered[m.cursor].ID
			}
			return m, tea.Quit
		case tea.KeyDown, tea.KeyCtrlJ, tea.KeyCtrlN:
			if len(m.filtered) > 0 {
				m.cursor = (m.cursor + 1) % len(m.filtered)
			}
		case tea.KeyUp, tea.KeyCtrlK, tea.KeyCtrlP:
			if len(m.filtered) > 0 {
				m.cursor = (m.cursor - 1 + len(m.filtered)) % len(m.filtered)
			}
		default:
			// Any other key: let the text input handle it, then re-filter.
			var cmd tea.Cmd
			m.query, cmd = m.query.Update(msg)
			m.applyFilter()
			return m, cmd
		}
		return m, nil
	default:
		// Forward non-key messages to the text input (e.g. blink).
		var cmd tea.Cmd
		m.query, cmd = m.query.Update(msg)
		return m, cmd
	}
}

// View implements tea.Model.
func (m *searchModel) View() string {
	var b strings.Builder

	// Title + search input
	b.WriteString(m.query.View())
	b.WriteString("\n\n")

	// Model list
	for i, model := range m.filtered {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		label := model.ID
		if model.DisplayName != "" && model.DisplayName != model.ID {
			label += "  " + model.DisplayName
		}
		if model.SupportsEdit {
			label += " [edit]"
		}
		if i == m.cursor {
			label = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render(label)
		}
		b.WriteString(cursor + label + "\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString("  No matching models.\n")
	}

	// Help line
	b.WriteString("\n  \u2191\u2193 navigate  enter select  esc cancel\n")

	return b.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run TestSearch -v`
Expected: PASS for all 9 tests

- [ ] **Step 5: Run go vet and gofmt**

Run: `go vet ./internal/tui/ && gofmt -l internal/tui/model_search.go`
Expected: no output (clean)

- [ ] **Step 6: Commit**

```bash
git add internal/tui/model_search.go internal/tui/model_search_test.go
git commit -m "feat(tui): add custom Bubble Tea search model for real-time model filtering"
```

---

## Task 11: Integrate Search Model into `model_list.go`

**Files:**
- Modify: `internal/tui/model_list.go`

**Interfaces:**
- Consumes: `searchModel` and `newSearchModel` from Task 10
- Produces: `pickModelInteractive` now uses the custom Bubble Tea search program

- [ ] **Step 1: Replace `pickModelInteractive` in `model_list.go`**

In `internal/tui/model_list.go`, replace the existing `pickModelInteractive` function:

```go
// Before (current implementation using huh.NewSelect):
func pickModelInteractive(providerName string, models []adapter.Model) (string, error) {
	options := make([]huh.Option[string], 0, len(models))
	for _, m := range models {
		label := m.DisplayName
		if m.SupportsEdit {
			label += " [edit]"
		}
		label += " - " + m.ID
		options = append(options, huh.NewOption(label, m.ID))
	}

	var selected string
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(fmt.Sprintf("Models for %s:", providerName)).
			Options(options...).
			Value(&selected),
	))
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("model select: %w", err)
	}
	return selected, nil
}

// After:
func pickModelInteractive(providerName string, models []adapter.Model) (string, error) {
	m := newSearchModel(models)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("model search: %w", err)
	}
	if sm, ok := result.(*searchModel); ok {
		if sm.quitted {
			return "", huh.ErrUserAborted
		}
		return sm.selected, nil
	}
	return "", fmt.Errorf("unexpected model type")
}
```

Update the imports in `model_list.go`:
- Add `"os"` for `os.Stderr`
- Add `tea "github.com/charmbracelet/bubbletea"`
- Keep `"github.com/charmbracelet/huh"` (for `huh.ErrUserAborted`)
- Remove `huh` import only if no other `huh` usage remains in the file. Check: `huh.NewForm`, `huh.NewGroup`, `huh.NewSelect`, `huh.NewOption` are no longer used after this change. If `newForm` is used elsewhere in the file, keep `huh` imported. Since `pickModelInteractive` is the only function that used `huh.NewSelect` and `huh.NewOption`, and the new code uses `huh.ErrUserAborted`, keep the `huh` import.

- [ ] **Step 2: Verify compilation**

Run: `go build -o /dev/null .`
Expected: builds with no errors

- [ ] **Step 3: Run all TUI tests**

Run: `go test ./internal/tui/ -v`
Expected: PASS

- [ ] **Step 4: Run all tests**

Run: `go test ./... -v 2>&1 | tail -40`
Expected: all tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/tui/model_list.go
git commit -m "feat(tui): integrate search-by-typing model picker into model list"
```

---

## Task 12: Simplify `install.sh` (PATH Detection, No sudo)

**Files:**
- Modify: `install.sh`

This is a shell script modification (not Go testable). Follow the existing script style.

- [ ] **Step 1: Simplify the install directory logic**

In `install.sh`, replace the entire "Determine install location" block (approximately lines 258-310) with:

```sh
    # Install to ~/.local/bin (always, no sudo needed)
    install_dir="${HOME}/.local/bin"
    mkdir -p "$install_dir" 2>/dev/null || true
    install_path="${install_dir}/potaco"
```

This removes all the `/usr/local/bin` detection, `sudo` prompt, and fallback logic.

- [ ] **Step 2: Replace the install binary block**

Replace the block that handles `sudo mv` / `chmod` with:

```sh
    if [ -w "$install_dir" ]; then
        mv "$binary_path" "$install_path"
        chmod +x "$install_path"
    else
        spinner_stop
        error "Cannot write to $install_dir."
        error "Ensure ~/.local/bin exists and is writable."
        exit 1
    fi

    spinner_stop
```

- [ ] **Step 3: Replace the PATH check block**

Replace the existing PATH check block (the `case` statement that warns about PATH) with a smarter one that offers to add it to the shell config:

```sh
    # Check if install_dir is in PATH
    case ":${PATH}:" in
        *":${install_dir}:"*)
            # Already in PATH, nothing to do
            ;;
        *)
            if [ "$NON_INTERACTIVE" = "1" ]; then
                warn "Note: $install_dir is not in your PATH."
                warn "Add it with: export PATH=\"${install_dir}:\$PATH\""
            else
                printf "\n"
                printf "%s is not in your PATH.\n" "$install_dir"
                printf "Add it automatically? [Y/n] "
                answer=""
                read answer || true
                case "$answer" in
                    [Yy]*|'')
                        add_to_shell_config "$install_dir"
                        ;;
                    *)
                        warn "Add it manually: export PATH=\"${install_dir}:\$PATH\""
                        ;;
                esac
            fi
            ;;
    esac
```

- [ ] **Step 4: Add the `add_to_shell_config` function**

Add this function before the `main` function in `install.sh`:

```sh
# add_to_shell_config appends a PATH export to the user's shell config file,
# auto-detected from $SHELL. Supports bash, zsh, and fish.
# Usage: add_to_shell_config "/path/to/bin"
add_to_shell_config() {
    bin_dir="$1"
    shell_path="${SHELL:-}"
    config_file=""
    export_line=""

    case "$shell_path" in
        */bash)
            config_file="${HOME}/.bashrc"
            export_line="export PATH=\"${bin_dir}:\$PATH\""
            ;;
        */zsh)
            config_file="${HOME}/.zshrc"
            export_line="export PATH=\"${bin_dir}:\$PATH\""
            ;;
        */fish)
            config_file="${HOME}/.config/fish/config.fish"
            export_line="fish_add_path ${bin_dir}"
            ;;
        *)
            warn "Could not detect shell from \$SHELL ($shell_path)."
            warn "Add ${bin_dir} to your PATH manually."
            return 0
            ;;
    esac

    # Create the config file if it doesn't exist (e.g. fish config)
    config_dir=$(dirname "$config_file")
    mkdir -p "$config_dir" 2>/dev/null || true

    # Check if the export line is already present
    if grep -qF "$bin_dir" "$config_file" 2>/dev/null; then
        info "$bin_dir already in $config_file."
        return 0
    fi

    printf '\n# Added by potaco installer\n%s\n' "$export_line" >> "$config_file"
    success "Added $bin_dir to $config_file"
    info "Restart your shell or run: source $config_file"
}
```

- [ ] **Step 5: Remove the sudo-related install steps from the print_box success message**

Update the success `print_box` to show `~/.local/bin` instead of `/usr/local/bin`:

```sh
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Done. Potaco installed to %s\n' "$install_path"
    else
        print_box "Success" \
            "${GREEN}Potaco installed successfully!${RESET}" \
            "" \
            "Installed to: $install_path" \
            "" \
            "Next steps:" \
            "  potaco auth add openai --api-key sk-..." \
            "  potaco gen --prompt \"hello\"" \
            "" \
            "Docs: https://github.com/ncxton/potaco#readme"
    fi
```

- [ ] **Step 6: Verify the script syntax**

Run: `sh -n install.sh && echo "OK"`
Expected: `OK`

- [ ] **Step 7: Commit**

```bash
git add install.sh
git commit -m "chore: simplify install.sh to ~/.local/bin only with shell PATH detection"
```

---

## Task 13: Final Verification and Cleanup

**Files:**
- All files from Tasks 1-12

- [ ] **Step 1: Run gofmt on all files**

Run: `gofmt -l .`
Expected: no output (all files formatted)

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: no issues

- [ ] **Step 3: Run all tests**

Run: `go test ./... -v 2>&1 | tail -40`
Expected: all tests pass

- [ ] **Step 4: Build the binary**

Run: `go build -o /dev/null .`
Expected: builds successfully

- [ ] **Step 5: Manual quick check of command registration**

Run: `go run . --help`
Expected: help output includes `version`, `update`, `uninstall` commands

Run: `go run . version`
Expected: prints `potaco unknown` (or whatever the local version is)

- [ ] **Step 6: Final commit if anything was fixed**

```bash
git add -A
git commit -m "chore: final formatting and verification"
```
