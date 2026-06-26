# CLI Utility Commands and TUI Enhancements Design

**Date:** 2026-06-26  
**Status:** Draft  
**Author:** Potaco Team

## Overview

This spec covers four new features for the Potaco CLI:

1. **New commands:** `version`, `update` (alias `upgrade`), and `uninstall`
2. **Interactive flow for `auth remove`** when called without arguments in a TTY
3. **Searchable model picker** for `potaco models` that filters as the user types
4. **Esc key cancels** any interactive TUI flow

## 1. Version Command (`potaco version`)

### Version Variable Injection

Add a `version` variable to `main.go` that defaults to `"unknown"` and is overridden by goreleaser via ldflags at build time.

**`main.go` changes:**
```go
package main

import "github.com/ncxton/potaco/internal/cli"

var version = "unknown"

func main() {
    cli.SetVersion(version)
    cli.Execute()
}
```

**`.goreleaser.yaml` changes:** Add `ldflags` to the build section:
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

Using `{{.Tag}}` so the version string includes the `v` prefix (e.g., `v1.2.3`).

### CLI Layer

**New file:** `internal/cli/version.go`

```go
package cli

var Version = "unknown"

func SetVersion(v string) {
    Version = v
}
```

**New file:** `internal/cli/version_cmd.go`

Registers a `version` subcommand on the root command.

### Behavior

- `potaco version` prints: `potaco v1.2.3 (latest: v1.3.0, update available)` or `potaco v1.2.3 (latest: v1.2.3, up to date)` or `potaco v1.2.3` (if GitHub API check fails)
- `potaco version --json` prints: `{"current": "v1.2.3", "latest": "v1.3.0", "update_available": true}`

### GitHub API Query

A function `checkLatestVersion()` performs an HTTP GET to `https://api.github.com/repos/ncxton/potaco/releases/latest` and parses the `tag_name` field from the JSON response.

**Caching:** The result is cached in a package-level variable with a timestamp. The cache is valid for 1 hour. Subsequent calls within the same process (e.g., `version` and `update` sharing the cache) reuse the cached value. Since the binary is short-lived (one command per invocation), this mainly avoids redundant calls within a single invocation where both `version` and `update` logic might query the API.

```go
var (
    latestCache     string
    latestCacheTime time.Time
    latestCacheErr  error
)

const latestCacheTTL = 1 * time.Hour

func checkLatestVersion() (string, error) {
    if time.Since(latestCacheTime) < latestCacheTTL && latestCache != "" {
        return latestCache, nil
    }
    // HTTP GET to GitHub API, parse tag_name
    // On success: update cache
    // On error: return ("", err) but don't cache errors
}
```

### Error Handling

- If the GitHub API call fails (network error, rate limit), print just the current version with no error. The `--json` output sets `latest` to `""` and `update_available` to `false`.
- Graceful degradation: never fail the version command due to a network issue.

### Testing

- `version_cmd_test.go`: Test that the command prints the version string (set `cli.Version` in tests). Mock the GitHub API with `httptest.Server`. Test JSON output format. Test cache behavior (two calls within the same test should only hit the server once within the TTL).

## 2. Update Command (`potaco update`, alias `upgrade`)

**New file:** `internal/cli/update_cmd.go`

### Flow

1. Start a spinner: "Checking for updates..."
2. Query the latest version via `checkLatestVersion()` (reuses the cached result from version command)
3. Compare current version (`cli.Version`) with latest:
   - If equal and `--force` is not set: print "Already up to date" and exit 0
   - If current is `"unknown"` (locally built): always proceed
4. Stop spinner
5. Start a new spinner: "Downloading installer..."
6. Download `install.sh` from `https://github.com/ncxton/potaco/releases/download/<tag>/install.sh`
7. Write to a temp file (`os.CreateTemp("", "potaco-install-*.sh")`) and `chmod +x`
8. Stop spinner
9. Execute the installer: `sh /tmp/potaco-install-XXXXX.sh`
   - Inherit stdin/stdout/stderr for interactivity
   - If `--non-interactive` is set, pass `POTACO_NON_INTERACTIVE=1` as an env var
10. Clean up the temp file (defer)
11. Exit with the installer's exit code

### Flags

- `--force` / `-f`: Re-run even if already at the latest version
- `--non-interactive` is inherited from the root persistent flag; passed to install.sh as `POTACO_NON_INTERACTIVE=1`

### Alias

`potaco upgrade` is an alias for `potaco update` (set via `Aliases: []string{"upgrade"}` on the cobra command).

### Error Handling

- GitHub API fails: `configUserErr("Could not check for updates.", "Check your network connection and try again.", err)`
- install.sh download fails: `apiUserErr("Could not download the installer.", "Check your network connection and try again.", err)`
- install.sh execution fails: pass through the error with a user-facing message

### Testing

- `update_cmd_test.go`: Test "already up to date" path by setting `cli.Version` to match a mock latest. Test `--force` bypasses the version check. Mock the GitHub API with `httptest.Server` for the version check. For the actual install.sh download/execute, test only the download step (mock the URL), not the actual execution (which would require a real shell).

## 3. Uninstall Command (`potaco uninstall`)

**New file:** `internal/cli/uninstall_cmd.go`

### Prerequisite: Simplify install.sh

Modify `install.sh` to always install to `$HOME/.local/bin` (no `/usr/local/bin`, no sudo). This ensures the binary is always in a user-writable location and simplifies uninstall.

**install.sh changes:**
- Remove the `/usr/local/bin` detection, sudo prompt, and related logic
- Always use `install_dir="${HOME}/.local/bin"` and `mkdir -p "$install_dir"`
- After installation, check if `$HOME/.local/bin` is in `$PATH`
- If not in PATH, detect the current shell via `$SHELL` environment variable:
  - `*/bash` -> append `export PATH="$HOME/.local/bin:$PATH"` to `~/.bashrc`
  - `*/zsh` -> append to `~/.zshrc`
  - `*/fish` -> append to `~/.config/fish/config.fish` (using `fish_add_path` syntax)
- Prompt the user: "Add $HOME/.local/bin to PATH? [Y/n]" (in interactive mode only)

### Uninstall Flow (Interactive)

1. Locate the binary via `os.Executable()` (resolves symlinks)
2. Show spinner briefly: "Locating potaco..."
3. Confirm prompt (huh): "This will remove the potaco binary at `<path>`. Continue?" (Yes/No)
   - If No: print "Cancelled." and exit 0
4. Confirm prompt (huh): "Also remove configuration and credentials at `~/.potaco/`?" (Yes/No)
5. Regardless of step 4 answer:
6. Final confirmation (huh): summary of what will be removed, e.g. "Confirm: remove binary [and config dir]?" (Yes/No)
   - If No: print "Cancelled." and exit 0
7. Execute removal:
   - Remove the binary file (`os.Remove(path)`)
   - If config removal was opted in: `os.RemoveAll(filepath.Join(home, ".potaco"))`
8. Print success message listing what was removed
9. Note: PATH entry in shell config is left intact (may be used by other tools)

### Uninstall Flow (Non-Interactive)

- `--non-interactive`: Remove binary only, skip prompts and config removal. Print what was done.
- `--remove-config`: Also remove config dir in non-interactive mode
- `--yes` / `-y`: Skip all confirmation prompts (auto-confirm), but still follow the interactive selection flow for config removal if in a TTY

### Flags

- `--remove-config`: Also remove `~/.potaco/` config directory
- `--yes` / `-y`: Skip confirmation prompts, auto-answer Yes to all prompts including config removal. In interactive mode, still shows the config removal prompt but pre-answers Yes.

### Error Handling

- Binary removal fails (permissions): `configUserErr("Cannot remove the binary.", "Check file permissions or remove manually: <path>", err)`
- Binary not found: warn but don't fail (print "Binary not found, may have been removed already.")
- Config dir removal fails: warn but don't fail the whole operation
- `os.Executable()` fails: fall back to searching `$PATH` for `potaco`

### Testing

- `uninstall_cmd_test.go`: Test non-interactive flow (binary removal only, binary + config). Test `--yes` flag skips prompts. Use a temp binary file instead of the real one (mock `os.Executable` or test the removal logic directly). Test error when binary path is not writable.

## 4. Interactive `auth remove` Flow

**New file:** `internal/tui/auth_remove.go`  
**Modified file:** `internal/cli/auth_cmd.go`

### Behavior Change

Currently, `auth remove <provider>` requires a provider arg (via `cobra.ExactArgs(1)`) and removes it directly. The command's `Args` constraint must change to `cobra.MaximumNArgs(1)` to allow zero-arg invocation. The new behavior:

**`potaco auth remove` (no args, interactive):**
1. Load connected providers via `mgr.List()`
2. If no providers: print "No providers connected." and exit 0
3. Show a `huh.NewSelect` picker listing connected providers (with `(active)` marker, model name, same as `use_picker.go`)
4. User selects a provider
5. Show `huh.NewConfirm`: "Remove provider '<name>' and its credentials?" (Yes/No)
6. If Yes: `mgr.Remove(providerName)`, print "Provider '<name>' removed."
7. If No: print "Cancelled."

**`potaco auth remove <provider>` (with args, interactive):**
- Skip the picker, go straight to confirmation (step 5)

**`potaco auth remove <provider>` (non-interactive):**
- Remove directly as before (no confirmation prompt)

**`potaco auth remove` (no args, non-interactive):**
- Error: "Specify a provider: potaco auth remove <provider>"

### TUI Implementation

`internal/tui/auth_remove.go`:
```go
// RunAuthRemove launches the interactive auth remove flow.
// If providerName is empty, shows a provider picker first.
// Shows a confirmation prompt before removing.
func RunAuthRemove(providerName string) error {
    // If providerName == "", show provider picker via huh.NewSelect
    // Then show confirmation prompt via huh.NewConfirm
    // If confirmed, call mgr.Remove(providerName)
    // If user cancels (Esc/Ctrl+C), print "Cancelled." and return nil
}
```

### Testing

- `auth_cmd_test.go` (existing): Add tests for interactive flow using `--non-interactive` to skip TUI. Test that `auth remove` without args in non-interactive mode returns error. The interactive TUI flow itself is smoke-tested (requires TTY).

## 5. Models Search-by-Typing

**Modified file:** `internal/tui/model_list.go`  
**New component:** Custom Bubble Tea program (uses `charmbracelet/bubbletea`, already an indirect dependency)

### Why Custom Bubble Tea

The `huh.NewSelect` field has a built-in `/` filter, but activating it requires pressing `/` first. The requirement is that typing any character immediately filters the list without a prefix key. The `huh` library doesn't expose a hook for "on any key press, enter filter mode." A custom Bubble Tea program gives full control over the keyboard input.

### Component Design

A new `searchModel` struct implementing `tea.Model`:

```go
type searchModel struct {
    models    []adapter.Model
    filtered  []adapter.Model
    cursor    int
    query     textinput.Model
    selected  string
    quitted   bool
}

func newSearchModel(models []adapter.Model) *searchModel
func (m *searchModel) Init() tea.Cmd
func (m *searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m *searchModel) View() string
```

### UI Layout

```
Models for openai:
> gpt-image-2_
  gpt-image-2           GPT Image 2 [edit]
  dall-e-3              DALL-E 3
  ...

  ↑↓ navigate  enter select  esc cancel
```

- The `>` prefix and underscore cursor indicate the text input is always focused
- The filtered list updates in real-time as the user types
- The list is case-insensitively filtered on both model ID and display name
- `↑`/`↓` (or `k`/`j`) navigate the filtered list
- `Enter` selects the highlighted model and returns its ID
- `Esc` cancels and returns an error (`huh.ErrUserAborted` or a custom cancel error)
- `Ctrl+C` also cancels

### Integration

`RunModelList` in `model_list.go` replaces the current `pickModelInteractive` (which uses `huh.NewSelect`) with the new custom search model:

```go
func pickModelInteractive(providerName string, models []adapter.Model) (string, error) {
    m := newSearchModel(models)
    p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
    result, err := p.Run()
    if err != nil {
        return "", fmt.Errorf("model search: %w", err)
    }
    if m, ok := result.(*searchModel); ok {
        if m.quitted {
            return "", huh.ErrUserAborted
        }
        return m.selected, nil
    }
    return "", fmt.Errorf("unexpected model type")
}
```

### Testing

- `model_list_test.go`: Test the filtering logic (case-insensitive match on ID and display name). Test that the cursor wraps or clamps. Test Esc returns abort error. These tests use direct struct method calls, not the full Bubble Tea program (which needs a TTY).

## 6. Esc Cancels Any Interactive TUI

### Approach

Override the global `KeyMap.Quit` binding on all `huh` forms to include both `ctrl+c` and `esc`.

**New helper in `internal/tui/tui.go`:**

```go
import "github.com/charmbracelet/huh"
import "github.com/charmbracelet/bubbles/key"

// newForm creates a huh.Form with the Esc key bound to quit alongside ctrl+c.
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
```

### Usage

All existing `huh.NewForm(...)` calls throughout the TUI package are replaced with `newForm(...)`:
- `auth_add.go`: 3 form creations (key input, verify confirm, model select)
- `use_picker.go`: 3 form creations (provider select, model confirm, model input)
- `auth_remove.go`: 2 form creations (provider select, removal confirm) (new file)

### Abort Handling

When the user presses Esc, `huh` returns `huh.ErrUserAborted` from `form.Run()`. Each TUI function checks for this:

```go
if err := form.Run(); err != nil {
    if errors.Is(err, huh.ErrUserAborted) {
        fmt.Println("Cancelled.")
        return nil
    }
    return fmt.Errorf("form: %w", err)
}
```

This pattern is applied consistently across all TUI flows. The CLI command layer treats a nil error as success (no exit code change).

### Custom Bubble Tea Program (models search)

The custom search model handles Esc directly in its `Update` method:

```go
case tea.KeyEsc:
    m.quitted = true
    return m, tea.Quit
```

### Dual-Meaning of Esc in huh Select

When a `huh.NewSelect` field is in filter mode (user pressed `/`), pressing `esc` first clears the filter (handled internally by huh). Pressing `esc` again (when not in filter mode) aborts the form. This is the accepted dual-meaning behavior.

## File Summary

### New Files
- `internal/cli/version.go` - Version variable and setter
- `internal/cli/version_cmd.go` - `version` subcommand
- `internal/cli/update_cmd.go` - `update` (alias `upgrade`) subcommand
- `internal/cli/uninstall_cmd.go` - `uninstall` subcommand
- `internal/tui/auth_remove.go` - Interactive auth remove flow
- `internal/cli/version_cmd_test.go`
- `internal/cli/update_cmd_test.go`
- `internal/cli/uninstall_cmd_test.go`

### Modified Files
- `main.go` - Add `version` variable, call `cli.SetVersion(version)`
- `.goreleaser.yaml` - Add `ldflags` to build section
- `install.sh` - Simplify to `~/.local/bin` only, add PATH detection
- `internal/cli/auth_cmd.go` - Interactive flow dispatch for `auth remove`
- `internal/tui/tui.go` - Add `newForm` helper with Esc key
- `internal/tui/auth_add.go` - Use `newForm` instead of `huh.NewForm`
- `internal/tui/use_picker.go` - Use `newForm` instead of `huh.NewForm`
- `internal/tui/model_list.go` - Custom Bubble Tea search program
- `internal/cli/auth_cmd_test.go` - Add tests for new auth remove behavior
- `internal/tui/model_list_test.go` - Add tests for search/filter logic

### Not Changed
- The adapter, auth, config, credential, and image packages remain untouched
- `internal/cli/root.go` may need a minor change if we want to show version in `--help` output (optional, not required)

## Dependencies

No new external dependencies. All required libraries are already in `go.mod`:
- `charmbracelet/bubbletea` (indirect, through `huh`) - for the custom search model
- `charmbracelet/bubbles/textinput` (indirect, through `huh`) - for the search input
- `charm.land/lipgloss/v2` - for styling (already used)

## Error Handling Summary

| Command | Error Scenario | Exit Code | User-Facing Message |
|---|---|---|---|
| version | GitHub API fails | 0 (graceful) | Print current version only |
| update | GitHub API fails | 2 (config) | "Could not check for updates" + hint |
| update | install.sh download fails | 3 (API) | "Could not download the installer" + hint |
| update | Already up to date | 0 | "Already up to date" |
| uninstall | Binary removal fails | 2 (config) | "Cannot remove the binary" + hint |
| uninstall | Binary not found | 0 | Warn, but don't fail |
| auth remove (TUI Esc) | User cancels | 0 | "Cancelled." |
| models (TUI Esc) | User cancels | 0 | "Cancelled." |

## Testing Strategy

- **Unit tests** for each new command using the existing test patterns (`rootCmd.SetArgs`, `rootCmd.Execute`, `httptest.Server` for mocking)
- **Non-interactive** paths are fully testable
- **Interactive TUI** flows are smoke-tested (they require a TTY for full testing)
- **Search model** filtering logic is tested via direct struct method calls
- **GitHub API** mocking via `httptest.Server`
- **install.sh** modifications are tested manually (shell script, not Go-testable)

## Open Questions

None. All design decisions have been resolved through the brainstorming process.
