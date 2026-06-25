# Phase 5: Bubbletea TUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add interactive TUI flows using the Charm ecosystem (Bubbletea v2, Bubbles v2, Lipgloss v2, Huh) for `auth add`, `use` picker, `models` list, and styled output for `auth list` and `status`, with `--non-interactive` global flag and TTY detection for automatic flow routing.

**Architecture:** A new `internal/tui/` package contains shared TUI helpers (TTY detection, launch helper, error display) and per-flow Bubbletea models. CLI commands check `tui.IsInteractive()` and either launch the TUI or fall through to the existing non-interactive path. The TUI layer calls back into `adapter` and `auth` packages for business logic. No business logic lives in the TUI layer. All existing non-interactive behavior from Phases 1-4 is preserved.

**Tech Stack:** Go 1.26, `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`, `github.com/charmbracelet/huh`, Cobra CLI

## Global Constraints

- Go 1.26, pure Go only (no CGO)
- Standard `gofmt` formatting. Run `gofmt -l .` before committing; fix any flagged files
- No panics in library code. Use `fmt.Errorf` with `%w` for error wrapping
- No `_ = err` (every `(T, error)` must be checked)
- `context.Context` as first param where applicable
- Keep files under 250 pure LOC
- Table-driven tests preferred. Test files sit alongside source: `foo.go` / `foo_test.go`
- CLI tests dispatch via `rootCmd.SetArgs([]string{...})` and `rootCmd.Execute()`
- Use `t.TempDir()` for temp files and `t.Setenv()` for env vars in tests
- Exit codes: 0 success, 2 config error, 3 API error, 4 image error
- Module path: `github.com/ncxton/potaco`
- All existing non-interactive behavior from Phases 1-4 MUST continue to work identically
- TUI flows are only launched when a TTY is detected AND `--non-interactive` is NOT set
- TUI models are tested via `teatest` or by calling `Update` and `View` with simulated messages
- The `--non-interactive` flag is a persistent flag on `rootCmd`
- TTY detection uses `os.Stdin` stat to check if stdin is a character device

---

## File Structure

### New Files

| File | Responsibility |
|------|----------------|
| `internal/tui/tui.go` | Shared helpers: `IsInteractive()`, `IsTTY()`, `LaunchError()`, error display helpers |
| `internal/tui/tui_test.go` | Tests for TTY detection and helpers |
| `internal/tui/auth_add.go` | Interactive `auth add` flow: key input, verify spinner, model picker |
| `internal/tui/auth_add_test.go` | Tests for auth add flow model |
| `internal/tui/use_picker.go` | Interactive `use` picker: provider list, model selection |
| `internal/tui/use_picker_test.go` | Tests for use picker model |
| `internal/tui/model_list.go` | Interactive `models` list: scrollable model list with badges |
| `internal/tui/model_list_test.go` | Tests for model list model |

### Modified Files

| File | Changes |
|------|---------|
| `go.mod` | Add Charm dependencies |
| `go.sum` | Updated by `go get` |
| `internal/cli/root.go` | Add `--non-interactive` persistent flag |
| `internal/cli/auth_cmd.go` | Check `tui.IsInteractive()` in `runAuthAdd`, route to TUI or non-interactive |
| `internal/cli/use_cmd.go` | Check `tui.IsInteractive()` in `runUse` (no args case), route to TUI or error |
| `internal/cli/models_cmd.go` | Check `tui.IsInteractive()` in `runModels`, route to TUI or text output |
| `internal/cli/status_cmd.go` | Apply lipgloss styling to text output when TTY |
| `internal/cli/auth_cmd.go` | Apply lipgloss styling to `auth list` output when TTY |

---

## Task 1: Add Charm Dependencies and TUI Foundation

**Files:**
- Modify: `go.mod`, `go.sum`
- Create: `internal/tui/tui.go`
- Create: `internal/tui/tui_test.go`
- Modify: `internal/cli/root.go` (add `--non-interactive` flag)

**Interfaces:**
- Consumes: `os`, `os/exec`, `github.com/spf13/cobra`
- Produces: `tui.IsInteractive() bool`, `tui.IsTTY() bool`, `--non-interactive` persistent flag on rootCmd

- [ ] **Step 1: Add Charm dependencies**

Run:
```bash
cd /home/ngct/Projects/potaco
go get charm.land/bubbletea/v2@latest
go get charm.land/bubbles/v2@latest
go get charm.land/lipgloss/v2@latest
go get github.com/charmbracelet/huh@latest
go mod tidy
```

Verify `go.mod` has the new dependencies.

- [ ] **Step 2: Write the failing tests**

Create `internal/tui/tui_test.go`:

```go
package tui

import (
	"os"
	"testing"
)

func TestIsInteractiveReturnsFalseInTestEnv(t *testing.T) {
	// In test environments, stdin is not a TTY
	if IsInteractive() {
		t.Error("IsInteractive() should return false when stdin is not a TTY")
	}
}

func TestIsInteractiveReturnsFalseWhenNonInteractiveEnv(t *testing.T) {
	t.Setenv("POTACO_NON_INTERACTIVE", "1")
	if IsInteractive() {
		t.Error("IsInteractive() should return false when POTACO_NON_INTERACTIVE=1")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/tui/ -v`
Expected: FAIL with "package not found"

- [ ] **Step 4: Write the implementation**

Create `internal/tui/tui.go`:

```go
// Package tui provides shared TUI helpers for interactive terminal flows.
package tui

import (
	"os"
)

// IsInteractive returns true when stdin is a TTY and the user has not
// opted out of interactive mode via the --non-interactive flag or the
// POTACO_NON_INTERACTIVE environment variable.
func IsInteractive() bool {
	if os.Getenv("POTACO_NON_INTERACTIVE") == "1" {
		return false
	}
	return IsTTY()
}

// IsTTY returns true when stdin is a terminal (character device).
func IsTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeChar) != 0
}
```

- [ ] **Step 5: Add --non-interactive flag to root.go**

In `internal/cli/root.go`, add to the `init()` function:

```go
rootCmd.PersistentFlags().Bool("non-interactive", false, "force non-interactive mode (skip TUI)")
```

Also update `tui.go` to check the flag. Since the TUI package cannot import the CLI package (circular dependency), the CLI will set the `POTACO_NON_INTERACTIVE` env var or the TUI will be called with a parameter. Instead, have the CLI check the flag and pass it to `tui.IsInteractive()` via a package-level variable:

Add to `tui.go`:
```go
// nonInteractive is set by the CLI when --non-interactive is passed.
var nonInteractive bool

// SetNonInteractive enables or disables non-interactive mode.
func SetNonInteractive(v bool) {
	nonInteractive = v
}

// IsInteractive returns true when stdin is a TTY and the user has not
// opted out of interactive mode.
func IsInteractive() bool {
	if nonInteractive {
		return false
	}
	if os.Getenv("POTACO_NON_INTERACTIVE") == "1" {
		return false
	}
	return IsTTY()
}
```

In `internal/cli/root.go`, update `Execute()` to call `tui.SetNonInteractive`:

```go
func Execute() {
	nonInteractive, _ := rootCmd.Flags().GetBool("non-interactive")
	tui.SetNonInteractive(nonInteractive)
	if err := rootCmd.Execute(); err != nil {
```

Wait, `Execute()` calls `rootCmd.Execute()` which is the same function. This creates infinite recursion. The current `Execute()` function is the public entry point. Let me check:

Actually, looking at root.go, `Execute()` calls `rootCmd.Execute()`. The `--non-interactive` flag is a persistent flag, so it's parsed during `rootCmd.Execute()`. We need to set `tui.SetNonInteractive` inside `rootCmd.PersistentPreRun` or check it in each command.

Better approach: Use `rootCmd.PersistentPreRunE` to set the non-interactive mode before any command runs:

```go
func init() {
	rootCmd.PersistentFlags().Bool("non-interactive", false, "force non-interactive mode (skip TUI)")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		ni, _ := cmd.Flags().GetBool("non-interactive")
		tui.SetNonInteractive(ni)
		return nil
	}
}
```

Add the import for `tui` in `root.go`:
```go
import "github.com/ncxton/potaco/internal/tui"
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v`
Run: `go test ./... -count=1`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum internal/tui/tui.go internal/tui/tui_test.go internal/cli/root.go
git commit -m "tui: add Charm dependencies, TTY detection, and --non-interactive flag"
```

---

## Task 2: Interactive auth add Flow

**Files:**
- Create: `internal/tui/auth_add.go`
- Create: `internal/tui/auth_add_test.go`
- Modify: `internal/cli/auth_cmd.go` (route to TUI when interactive)

**Interfaces:**
- Consumes: `tui.IsInteractive()`, `adapter.Get()`, `adapter.Adapter.Verify()`, `adapter.Adapter.DiscoverModels()`, `auth.New()`, `auth.AuthManager.Add()`, `auth.AuthManager.SetActiveProvider()`
- Produces: `tui.RunAuthAdd(providerName string) error`

- [ ] **Step 1: Write the failing tests**

Create `internal/tui/auth_add_test.go`:

```go
package tui

import (
	"testing"
)

func TestAuthAddModelInit(t *testing.T) {
	m := newAuthAddModel("openai")
	if m.provider != "openai" {
		t.Errorf("provider = %q, want 'openai'", m.provider)
	}
	if m.state != authAddStatePrompt {
		t.Errorf("initial state = %v, want %v", m.state, authAddStatePrompt)
	}
}

func TestAuthAddModelCancelOnQuit(t *testing.T) {
	m := newAuthAddModel("openai")
	// Simulate 'q' key press to quit
	updated, _ := m.update(tea.QuitMsg{})
	if updated.(*authAddModel).state != authAddStateDone {
		t.Error("model should be done after quit")
	}
}
```

Note: Import `tea "charm.land/bubbletea/v2"` in the test file. The exact `tea` types will need to match the v2 API. Use `tea.QuitMsg` or the v2 equivalent. Check the v2 API docs for the correct message types.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestAuthAdd -v`
Expected: FAIL with "undefined: newAuthAddModel"

- [ ] **Step 3: Write the implementation**

Create `internal/tui/auth_add.go`:

This is the most complex TUI flow. It has states for: prompt for API key, verify spinner, verify failed confirm, discover spinner, model picker, and success. The model is a Bubbletea `Model` implementing `Init()`, `Update()`, and `View()`.

The implementation should:
1. Use `huh` form for the API key input (masked textinput)
2. Use a spinner while calling `Verify()`
3. Use `huh` confirm if verification fails
4. Use a spinner while calling `DiscoverModels()`
5. Use `bubbles/list` for model selection
6. On selection, call `auth.AuthManager.Add()` and `SetActiveProvider()`

Keep the file focused. If the model grows beyond 250 LOC, split the state handling into a separate file.

```go
package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
)

type authAddState int

const (
	authAddStatePrompt authAddState = iota
	authAddStateVerify
	authAddStateVerifyFailed
	authAddStateDiscover
	authAddStateModelPicker
	authAddStateSuccess
	authAddStateError
	authAddStateDone
)

type authAddModel struct {
	provider string
	apiKey   string
	state    authAddState
	// ... fields for sub-components (huh form, spinner, list, error message)
}

func newAuthAddModel(provider string) *authAddModel {
	return &authAddModel{
		provider: provider,
		state:    authAddStatePrompt,
	}
}

func (m *authAddModel) Init() tea.Cmd {
	return nil
}

func (m *authAddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case authAddStatePrompt:
		// Handle API key input from huh form
		// On submit, transition to authAddStateVerify
	case authAddStateVerify:
		// Handle spinner completion
		// On success, transition to authAddStateDiscover
		// On failure, transition to authAddStateVerifyFailed
	case authAddStateVerifyFailed:
		// Handle confirm prompt
		// On "yes", transition to authAddStateDiscover (skip verify)
		// On "no", transition to authAddStateDone (cancel)
	case authAddStateDiscover:
		// Handle spinner completion
		// On success, transition to authAddStateModelPicker
	case authAddStateModelPicker:
		// Handle list selection
		// On selection, store credential and transition to authAddStateSuccess
	case authAddStateSuccess:
		// Show success message, wait for any key to quit
	case authAddStateError:
		// Show error, wait for any key to quit
	}
	return m, nil
}

func (m *authAddModel) View() string {
	switch m.state {
	case authAddStatePrompt:
		return fmt.Sprintf("Enter API key for %s:", m.provider)
	case authAddStateVerify:
		return "Verifying..."
	case authAddStateVerifyFailed:
		return "Verification failed. Add anyway? (y/n)"
	case authAddStateDiscover:
		return "Discovering models..."
	case authAddStateModelPicker:
		return "Select a model:"
	case authAddStateSuccess:
		return fmt.Sprintf("Provider '%s' added successfully!", m.provider)
	case authAddStateError:
		return fmt.Sprintf("Error: %s", m.errMsg)
	default:
		return ""
	}
}

// RunAuthAdd launches the interactive auth add flow.
func RunAuthAdd(providerName string) error {
	// Implementation: create the model, run the Bubbletea program,
	// handle the result.
	// For now, fall through to non-interactive if the TUI fails.
	return nil
}
```

The above is a skeleton. The actual implementation needs to wire up `huh` forms, spinners, and lists. Each sub-component should be initialized in the state's init function. Keep the file under 250 LOC by splitting if needed.

IMPORTANT: The full implementation with `huh` forms and `bubbles/list` is complex. Start with a simplified version:
1. Read API key from a `huh` text input (masked)
2. Call `Verify()` directly (no spinner for the first pass)
3. If verify fails, use `huh` confirm
4. Call `DiscoverModels()` directly
5. Use `huh` select for model choice (simpler than `bubbles/list`)
6. Store credential

This simplified version uses `huh` for all interactions, which is much simpler than mixing Bubbletea models with huh. The `huh` library handles the TUI loop internally.

Revised approach - use `huh` forms for everything:

```go
package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
)

// RunAuthAdd launches the interactive auth add flow using huh forms.
func RunAuthAdd(providerName string) error {
	var apiKey string

	// Step 1: Prompt for API key
	keyForm := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title(fmt.Sprintf("Enter API key for %s:", providerName)).
			EchoMode(huh.EchoModePassword).
			Value(&apiKey),
	))
	if err := keyForm.Run(); err != nil {
		return fmt.Errorf("key input: %w", err)
	}
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	// Step 2: Verify the provider
	ad, err := adapter.Get(providerName, apiKey, adapter.AdapterOpts{})
	if err != nil {
		return fmt.Errorf("create adapter: %w", err)
	}

	verifyErr := ad.Verify(context.Background())

	// Step 3: If verification failed, ask to proceed
	if verifyErr != nil {
		var proceed bool
		confirmForm := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Verification failed: %s\nAdd anyway?", verifyErr)).
				Value(&proceed),
		))
		if err := confirmForm.Run(); err != nil {
			return fmt.Errorf("confirm: %w", err)
		}
		if !proceed {
			return fmt.Errorf("cancelled by user")
		}
	}

	// Step 4: Discover models
	models, discoverErr := ad.DiscoverModels(context.Background())

	// Step 5: Select model
	modelID := ""
	if discoverErr == nil && len(models) > 0 {
		options := make([]huh.Option[string], 0, len(models))
		for _, m := range models {
			label := m.DisplayName
			if m.SupportsEdit {
				label += " (supports edit)"
			}
			options = append(options, huh.NewOption(label, m.ID))
		}
		selectForm := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a model:").
				Options(options...).
				Value(&modelID),
		))
		if err := selectForm.Run(); err != nil {
			return fmt.Errorf("model select: %w", err)
		}
	}

	// Step 6: Store credential
	mgr, err := auth.New()
	if err != nil {
		return fmt.Errorf("init auth: %w", err)
	}
	if err := mgr.Add(providerName, apiKey, true); err != nil {
		return fmt.Errorf("add provider: %w", err)
	}
	if modelID != "" {
		if err := mgr.SetActiveProvider(providerName, modelID); err != nil {
			return fmt.Errorf("set model: %w", err)
		}
	}

	fmt.Printf("Provider '%s' added successfully.\n", providerName)
	return nil
}
```

This approach is much simpler and keeps the file under 100 LOC. The `huh` library handles the interactive TUI loop. No need for manual Bubbletea model management.

- [ ] **Step 4: Wire up the CLI routing**

In `internal/cli/auth_cmd.go`, modify `runAuthAdd` to check for interactive mode:

```go
func runAuthAdd(cmd *cobra.Command, args []string) error {
	providerName := args[0]

	// Check provider is a known adapter.
	available := adapter.List()
	// ... existing validation code ...

	// Interactive mode: launch TUI flow
	if tui.IsInteractive() {
		return tui.RunAuthAdd(providerName)
	}

	// Non-interactive mode: existing code path
	apiKey, _ := cmd.Flags().GetString("api-key")
	// ... rest of existing non-interactive code ...
}
```

Add the import:
```go
import "github.com/ncxton/potaco/internal/tui"
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v`
Run: `go test ./... -count=1`
Expected: PASS (TUI tests test the model structure, not the interactive flow)

- [ ] **Step 6: Commit**

```bash
git add internal/tui/auth_add.go internal/tui/auth_add_test.go internal/cli/auth_cmd.go
git commit -m "tui: add interactive auth add flow with huh forms"
```

---

## Task 3: Interactive use Picker

**Files:**
- Create: `internal/tui/use_picker.go`
- Create: `internal/tui/use_picker_test.go`
- Modify: `internal/cli/use_cmd.go` (route to TUI when interactive and no args)

**Interfaces:**
- Consumes: `tui.IsInteractive()`, `auth.New()`, `auth.AuthManager.List()`, `auth.AuthManager.SetActiveProvider()`
- Produces: `tui.RunUsePicker() error`

- [ ] **Step 1: Write the failing tests**

Create `internal/tui/use_picker_test.go`:

```go
package tui

import (
	"testing"
)

func TestRunUsePickerNoProvidersReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := RunUsePicker()
	if err == nil {
		t.Fatal("expected error when no providers connected")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestRunUsePicker -v`
Expected: FAIL with "undefined: RunUsePicker"

- [ ] **Step 3: Write the implementation**

Create `internal/tui/use_picker.go`:

```go
package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/ncxton/potaco/internal/auth"
)

// RunUsePicker launches the interactive provider/model picker.
func RunUsePicker() error {
	mgr, err := auth.New()
	if err != nil {
		return fmt.Errorf("init auth: %w", err)
	}

	providers := mgr.List()
	if len(providers) == 0 {
		return fmt.Errorf("no providers connected. Use 'potaco auth add <provider>' to connect one")
	}

	// Step 1: Select provider
	providerName := ""
	providerOptions := make([]huh.Option[string], 0, len(providers))
	for _, p := range providers {
		label := p.Name
		if p.IsActive {
			label += " (active)"
		}
		label += " - " + p.Model
		providerOptions = append(providerOptions, huh.NewOption(label, p.Name))
	}

	providerForm := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select a provider:").
			Options(providerOptions...).
			Value(&providerName),
	))
	if err := providerForm.Run(); err != nil {
		return fmt.Errorf("provider select: %w", err)
	}

	// Step 2: Select model (optional - can keep current)
	var changeModel bool
	confirmForm := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("Change the model for this provider?").
			Value(&changeModel),
	))
	if err := confirmForm.Run(); err != nil {
		return fmt.Errorf("confirm model change: %w", err)
	}

	modelID := ""
	if changeModel {
		// Discover models for the selected provider
		mgr.SetActiveProvider(providerName, "")
		provider, currentModel, _ := mgr.GetActiveProvider()
		_ = provider
		_ = currentModel

		// Use the provider's existing model as default
		for _, p := range providers {
			if p.Name == providerName {
				modelID = p.Model
				break
			}
		}

		modelForm := huh.NewForm(huh.NewGroup(
			huh.NewInput().
				Title("Enter model ID:").
				Value(&modelID),
		))
		if err := modelForm.Run(); err != nil {
			return fmt.Errorf("model input: %w", err)
		}
	}

	// Step 3: Set active provider
	if err := mgr.SetActiveProvider(providerName, modelID); err != nil {
		return fmt.Errorf("set active provider: %w", err)
	}

	fmt.Printf("Switched to provider '%s'.\n", providerName)
	return nil
}
```

- [ ] **Step 4: Wire up the CLI routing**

In `internal/cli/use_cmd.go`, modify `runUse` to check for interactive mode when no args:

```go
func runUse(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		if tui.IsInteractive() {
			return tui.RunUsePicker()
		}
		return configError(fmt.Errorf("specify a provider: potaco use <provider>"))
	}
	// ... existing code for use <provider> ...
}
```

Add import:
```go
import "github.com/ncxton/potaco/internal/tui"
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v`
Run: `go test ./... -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/use_picker.go internal/tui/use_picker_test.go internal/cli/use_cmd.go
git commit -m "tui: add interactive use picker with huh forms"
```

---

## Task 4: Interactive models List

**Files:**
- Create: `internal/tui/model_list.go`
- Create: `internal/tui/model_list_test.go`
- Modify: `internal/cli/models_cmd.go` (route to TUI when interactive)

**Interfaces:**
- Consumes: `tui.IsInteractive()`, `adapter.Get()`, `adapter.Adapter.DiscoverModels()`, `auth.New()`, `auth.AuthManager.GetActiveProvider()`, `auth.AuthManager.GetActiveAPIKey()`
- Produces: `tui.RunModelList(providerName, apiKey string) error`

- [ ] **Step 1: Write the failing tests**

Create `internal/tui/model_list_test.go`:

```go
package tui

import (
	"testing"
)

func TestRunModelListNoActiveProviderReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := RunModelList("", "")
	if err == nil {
		t.Fatal("expected error when no active provider")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestRunModelList -v`
Expected: FAIL with "undefined: RunModelList"

- [ ] **Step 3: Write the implementation**

Create `internal/tui/model_list.go`:

```go
package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
)

// RunModelList launches the interactive model list for the given provider.
// If providerName is empty, uses the active provider.
func RunModelList(providerName, apiKey string) error {
	mgr, err := auth.New()
	if err != nil {
		return fmt.Errorf("init auth: %w", err)
	}

	if providerName == "" {
		providerName, _, err = mgr.GetActiveProvider()
		if err != nil || providerName == "" {
			return fmt.Errorf("no active provider. Use 'potaco auth add <provider>' to connect one")
		}
	}

	if apiKey == "" {
		k, kErr := mgr.GetActiveAPIKey()
		if kErr == nil {
			apiKey = k
		}
	}
	if apiKey == "" {
		return fmt.Errorf("provider %q is not connected. Use 'potaco auth add %s' first", providerName, providerName)
	}

	ad, err := adapter.Get(providerName, apiKey, adapter.AdapterOpts{})
	if err != nil {
		return fmt.Errorf("create adapter: %w", err)
	}

	models, err := ad.DiscoverModels(context.Background())
	if err != nil {
		return fmt.Errorf("discover models: %w", err)
	}
	if len(models) == 0 {
		return fmt.Errorf("no models found for %s", providerName)
	}

	// Display models using huh select
	selected := ""
	options := make([]huh.Option[string], 0, len(models))
	for _, m := range models {
		label := m.DisplayName
		if m.SupportsEdit {
			label += " [edit]"
		}
		label += " - " + m.ID
		options = append(options, huh.NewOption(label, m.ID))
	}

	selectForm := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(fmt.Sprintf("Models for %s:", providerName)).
			Options(options...).
			Value(&selected),
	))
	if err := selectForm.Run(); err != nil {
		return fmt.Errorf("model select: %w", err)
	}

	// Show params for the selected model
	params, err := ad.ModelParams(context.Background(), selected)
	if err == nil && len(params) > 0 {
		fmt.Println("\nParameters:")
		for _, p := range params {
			fmt.Printf("  %s (%s) - %s (default: %s)\n", p.Name, p.Type, p.Description, p.Default)
		}
	}

	fmt.Printf("\nSelected: %s\n", selected)
	return nil
}
```

- [ ] **Step 4: Wire up the CLI routing**

In `internal/cli/models_cmd.go`, modify `runModels` to check for interactive mode:

```go
func runModels(cmd *cobra.Command, args []string) error {
	// ... existing setup code for providerName, apiKey, adapter ...

	// Check --params flag
	modelID := flagString(cmd, "params")
	if modelID != "" {
		return showModelParams(cmd, ad, modelID)
	}

	// Interactive mode: launch TUI
	if tui.IsInteractive() {
		return tui.RunModelList(providerName, apiKey)
	}

	// Non-interactive: existing text/JSON output
	models, err := ad.DiscoverModels(context.Background())
	// ... existing code ...
}
```

Add import:
```go
import "github.com/ncxton/potaco/internal/tui"
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v`
Run: `go test ./... -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/model_list.go internal/tui/model_list_test.go internal/cli/models_cmd.go
git commit -m "tui: add interactive model list with huh select"
```

---

## Task 5: Styled Output for status and auth list

**Files:**
- Modify: `internal/cli/status_cmd.go` (apply lipgloss styling when TTY)
- Modify: `internal/cli/auth_cmd.go` (apply lipgloss styling to list output when TTY)

**Interfaces:**
- Consumes: `tui.IsTTY()`, `charm.land/lipgloss/v2`
- Produces: Styled text output for status and auth list when running in a terminal

- [ ] **Step 1: Write the failing tests**

Add to `internal/cli/status_cmd_test.go`:

```go
func TestStatusStyledOutputContainsHeaders(t *testing.T) {
	setupAuthProviderForProvider(t, "openai", "sk-test", "gpt-image-2")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	output := buf.String()
	// Even without a TTY, the output should contain the key information
	if !strings.Contains(output, "Active provider") {
		t.Errorf("status should show active provider header, got: %s", output)
	}
}
```

Note: Since tests run without a TTY, the styling code path won't be exercised in tests. The test verifies the non-styled output still works. The styled path is tested manually.

- [ ] **Step 2: Add lipgloss styling to status_cmd.go**

In `internal/cli/status_cmd.go`, add lipgloss styles for the text output:

```go
import (
	"charm.land/lipgloss/v2"
	"github.com/ncxton/potaco/internal/tui"
)

var (
	statusTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	statusLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	statusActiveStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
)
```

Modify `runStatus` to use styles when TTY:

```go
func runStatus(cmd *cobra.Command, args []string) error {
	// ... existing setup ...

	if provider == "" {
		fmt.Fprintln(out, "No active provider configured.")
		fmt.Fprintln(out, "Use 'potaco auth add <provider>' to connect one.")
	} else {
		if tui.IsTTY() {
			fmt.Fprintf(out, "%s %s\n", statusLabelStyle.Render("Active provider:"), statusActiveStyle.Render(provider))
			fmt.Fprintf(out, "%s %s\n", statusLabelStyle.Render("Active model:"), model)
		} else {
			fmt.Fprintf(out, "Active provider: %s\n", provider)
			fmt.Fprintf(out, "Active model:    %s\n", model)
		}
	}
	// ... rest of output ...
}
```

- [ ] **Step 3: Add lipgloss styling to auth_cmd.go list output**

In `internal/cli/auth_cmd.go`, modify `runAuthList` to use lipgloss when TTY:

```go
import (
	"charm.land/lipgloss/v2"
	"github.com/ncxton/potaco/internal/tui"
)

func runAuthList(cmd *cobra.Command, args []string) error {
	// ... existing setup ...

	if tui.IsTTY() && !jsonMode {
		// Styled output
		titleStyle := lipgloss.NewStyle().Bold(true)
		activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
		keyOkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		keyMissingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))

		fmt.Fprintln(out, titleStyle.Render("Connected providers:"))
		fmt.Fprintln(out)
		for _, p := range providers {
			name := p.Name
			if p.IsActive {
				name = activeStyle.Render(p.Name + " (active)")
			}
			keyStatus := keyOkStyle.Render("configured")
			if !p.HasKey {
				keyStatus = keyMissingStyle.Render("missing")
			}
			fmt.Fprintf(out, "  %s  %s  key: %s\n", name, p.Model, keyStatus)
		}
		return nil
	}

	// ... existing non-styled output ...
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./... -count=1`
Expected: PASS (tests run without TTY, so non-styled path is tested)

- [ ] **Step 5: Commit**

```bash
git add internal/cli/status_cmd.go internal/cli/auth_cmd.go internal/cli/status_cmd_test.go
git commit -m "tui: add lipgloss styling for status and auth list output"
```

---

## Task 6: Final Verification

**Files:**
- No new files. Verification only.

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: All tests pass

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: No issues

- [ ] **Step 3: Run gofmt**

Run: `gofmt -l .`
Expected: No output

- [ ] **Step 4: Verify build**

Run: `go build -o potaco .`
Expected: Success

- [ ] **Step 5: Smoke test - non-interactive still works**

```bash
rm -rf /tmp/potaco-test
HOME=/tmp/potaco-test ./potaco auth add openai --api-key sk-test --force
HOME=/tmp/potaco-test ./potaco status
HOME=/tmp/potaco-test ./potaco models --params gpt-image-2
HOME=/tmp/potaco-test ./potaco use openai
HOME=/tmp/potaco-test ./potaco gen --prompt "a cat" --dry-run
```
Expected: All non-interactive commands work as before

- [ ] **Step 6: Smoke test - --non-interactive flag**

```bash
HOME=/tmp/potaco-test ./potaco auth add fal --api-key fal-key --force --non-interactive
HOME=/tmp/potaco-test ./potaco status --non-interactive
```
Expected: Commands run in non-interactive mode

- [ ] **Step 7: Check all file LOCs are under 250**

Run:
```bash
for f in internal/tui/*.go internal/cli/status_cmd.go internal/cli/auth_cmd.go internal/cli/models_cmd.go internal/cli/use_cmd.go; do
  loc=$(awk '!/^[[:space:]]*$/ && !/^[[:space:]]*(\/\/)/' "$f" | wc -l)
  echo "$loc  $f"
done
```
Expected: All files under 250 pure LOC

- [ ] **Step 8: Commit any final fixes if needed**

If any issues were found and fixed:
```bash
git add -A
git commit -m "fix: final verification fixes for Phase 5"
```

Otherwise, no commit needed.
