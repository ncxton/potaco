# Repository Guidelines

Potaco is a Go CLI for image generation and editing via multi-provider adapters (OpenAI, fal, Vercel AI Gateway) with encrypted credential storage and interactive TUI flows.

## Project Structure & Module Organization

```
main.go                  Entry point, calls cli.Execute()
internal/
  cli/                   Cobra commands and CLI infrastructure
    root.go              Root command, persistent flags (--json, --verbose, --non-interactive)
    gen.go               gen subcommand (text-to-image)
    edit.go, edit_mask.go  edit subcommand (image editing, inpainting, outpainting)
    auth_cmd.go          auth add/remove/list subcommands
    config_cmd.go        config set/show subcommands
    models_cmd.go        models subcommand (discover models, show params)
    status_cmd.go        status subcommand
    use_cmd.go           use subcommand (switch active provider)
    info.go              info subcommand (image metadata)
    resolve.go           Provider/credential/model resolution with flag>env>config precedence
    helpers.go           Flag accessors, provider presets, dry-run output, processAndOutput
    output.go            Output formatting (text, JSON, stdout modes)
    usererr.go           UserError type with friendly message, hint, debug logging
    errors.go            Exit code constants and legacy error wrappers
    spinner.go           Terminal spinner for gen/edit operations
  adapter/               Provider adapter interface and registry
    adapter.go           Adapter interface (Generate, Edit, DiscoverModels, Verify, ModelParams)
    registry.go          Factory registry: Register/Get/List for provider adapters
    openai/              OpenAI adapter (Images API, /v1/images/generations, /v1/images/edits)
    fal/                 fal adapter (fal.run inference, api.fal.ai discovery, image-to-image)
    vercel/              Vercel AI Gateway adapter (generate-only, no edit support)
  auth/                  AuthManager: coordinates credential store and multi-provider config
  credential/            Encrypted credential storage (AES-256-GCM, machine-derived key)
    store.go             CredentialStore: Get/Set/Remove/List API keys
    encrypt.go           Key derivation (hostname+username+salt), AES-256-GCM encrypt/decrypt
    types.go             ProviderCredential struct
  config/                Multi-provider YAML config (~/.potaco/config.yaml)
    config.go            Load/Save MultiProviderConfig, default path helpers
    types.go             MultiProviderConfig, ProviderConfig structs
  tui/                   Interactive terminal flows (huh forms, lipgloss styling)
    tui.go               IsInteractive/IsTTY/NonInteractive mode detection
    auth_add.go          Interactive auth add flow (key prompt, verify, model picker)
    model_list.go        Interactive model list and picker
    use_picker.go        Interactive provider/model switcher
  image/                 Image I/O, mask generation, outpaint canvas
    io.go                Read/decode (PNG, JPEG, WebP), write, base64 decode, auto-filename
    mask.go              RectMask, CircleMask, LoadMaskFile, WriteMask
    canvas.go            ParseExtend, PrepareOutpaint (outpaint canvas expansion)
docs/superpowers/        Design specs and implementation plans
  specs/                Design documents
  plans/                Phase-by-phase implementation plans
```

Layered monolith dependency graph:

```
cli --> adapter, auth, config, credential, tui, image
tui  --> adapter, auth
auth --> config, credential
adapter/openai|fal|vercel --> adapter (parent), config
config, credential, image --> (no internal deps)
```

All packages live under `internal/` (not importable externally). Adapter sub-packages register themselves via `init()` and are imported for side effects in `cli/helpers.go`.

## Build, Test, and Development Commands

```
go build -o potaco .       # Build the binary
go test ./...              # Run all tests
go test ./... -v           # Run all tests with verbose output
go vet ./...               # Lint
gofmt -l .                 # Check formatting (should output nothing)
```

Config file lives at `~/.potaco/config.yaml`. Encrypted credentials at `~/.potaco/credentials.enc` with salt at `~/.potaco/.salt`. For local testing without a real provider, use `--dry-run`:
```
POTACO_BASE_URL=https://api.openai.com POTACO_API_KEY=sk-test \
  ./potaco gen --prompt "a cat" --dry-run
```

Provider credentials are managed via `auth add`:
```
./potaco auth add openai --api-key sk-...   # Connect a provider
./potaco auth list                          # List connected providers
./potaco use openai                         # Switch active provider
./potaco models                             # Discover available models (interactive)
./potaco status                             # Show current provider/model status
```

## Coding Style & Naming Conventions

- Go 1.26, pure Go only (no CGO).
- Standard `gofmt` formatting. Run `gofmt -l .` before committing; fix any flagged files.
- Key dependencies: `spf13/cobra` (CLI), `charmbracelet/huh` (TUI forms), `charm.land/lipgloss/v2` (styling), `golang.org/x/image` (WebP decode), `golang.org/x/term` (TTY detection), `gopkg.in/yaml.v3` (config).
- Internal image package is imported as `img` in CLI files to avoid collision with stdlib `image`.
- No panics in library code. Use `fmt.Errorf` with `%w` for error wrapping.
- Keep files focused: one responsibility per file, one subcommand per file in `cli/`.
- Adapter providers register via `init()` calling `adapter.Register(name, factory)`. New providers are added as a sub-package under `internal/adapter/` implementing the `adapter.Adapter` interface, then blank-imported in `cli/helpers.go`.
- User-facing errors use `UserError` (via `configUserErr`, `apiUserErr`, `imageUserErr`) with a friendly message, optional hint, and raw error for debug logging. Legacy `configError`/`apiError`/`imageError` wrappers still exist for backward compatibility.
- Table-driven tests are preferred. Test files sit alongside source: `foo.go` / `foo_test.go`.

## Testing Guidelines

- Testing framework: Go standard `testing` package with `httptest.Server` for adapter/provider tests.
- TDD: write failing tests first, implement to pass, then commit.
- CLI tests dispatch via `rootCmd.SetArgs([]string{"subcommand", ...})` and `rootCmd.Execute()` (not subcommand-direct `Execute()`).
- Adapter tests use `httptest.Server` mocks and override `Adapter.backoff` and `Adapter.sleep` (via `SetBackoff`/`SetSleep`) to 1ms for fast retry tests.
- Credential tests verify encrypt/decrypt roundtrips with test keys and temp directories.
- Image tests use `t.TempDir()` for temp files and `bytes.Buffer` for in-memory roundtrips.
- TUI tests are minimal (smoke tests) since interactive forms require a TTY.

## Commit & Pull Request Guidelines

Conventional Commits format with scope matching the package or feature area:
```
feat(adapter): add provider adapter interface and registry
feat(tui): add interactive auth add flow with model picker
feat(cli): add auth, config, status, use, models subcommands
feat(credential): add encrypted credential storage with AES-256-GCM
fix(adapter): retry body reset and base URL handling for /v1 suffix
fix(cli): silence duplicated error output and exit codes
refactor(config): migrate to multi-provider YAML config format
```

Subject line is lowercase, no period. Use `feat(scope):` for new features, `fix(scope):` for bug fixes, `refactor(scope):` for restructuring, `docs:` for documentation, `chore:` for maintenance. The scope should match the package name (`adapter`, `cli`, `config`, `credential`, `auth`, `tui`, `image`) or a logical feature area.

**NEVER commit files under `docs/superpowers/` (specs, plans, or any other superpowers artifacts).** This directory is gitignored for a reason — these are local design documents, not shipped code. Do not use `git add -f` to force-add them.

## Exit Codes

Defined in `internal/cli/errors.go`: 0 success, 2 config error, 3 API error, 4 image error. New code should use `configUserErr`, `apiUserErr`, `imageUserErr` constructors from `usererr.go` which produce `UserError` values with a friendly message and hint. Legacy `configError()`, `apiError()`, `imageError()` wrappers still exist for backward compatibility. `Execute()` in `root.go` renders the error (colored on TTY, plain text otherwise) and writes the raw error to `~/.potaco/debug.log` before exiting with the appropriate code.
