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
    update_cmd.go        update subcommand (download and run latest release installer)
    version_cmd.go       version subcommand (print version, check for updates)
    version.go           Version variable, SetVersion (ldflags injection)
    uninstall_cmd.go     uninstall subcommand (remove binary and config)
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
      openai.go          Adapter struct, AuthHeader, Name
      generate.go        Generate (text-to-image)
      edit.go            Edit (inpainting with mask)
      discover.go        DiscoverModels (GET /v1/models)
      models.go          Fallback model list, ModelParams
      response.go        Response types
      retry.go           Retry with exponential backoff
    fal/                 fal adapter (fal.run inference, api.fal.ai discovery, image-to-image)
      fal.go             Adapter struct, AuthHeader, Name
      generate.go        Generate (text-to-image)
      edit.go            Edit (image-to-image)
      discover.go        DiscoverModels (POST to api.fal.ai)
      models.go          Fallback model list, ModelParams
      response.go        Response types
      retry.go           Retry with exponential backoff
    vercel/              Vercel AI Gateway adapter (generate-only, no edit support)
      vercel.go          Adapter struct, AuthHeader, Name
      generate.go        Generate (text-to-image)
      edit.go            Edit (returns ErrEditNotSupported)
      discover.go        DiscoverModels
      models.go          Fallback model list, ModelParams
      response.go        Response types
      retry.go           Retry with exponential backoff
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
    auth_remove.go       Interactive auth remove flow (provider picker, confirm)
    model_list.go        Interactive model list and picker
    model_search.go      Bubble Tea search model for real-time model filtering
    use_picker.go        Interactive provider/model switcher
  image/                 Image I/O, mask generation, outpaint canvas
    init.go              Side-effect import: register stdlib PNG/JPEG decoders
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

## Code Quality Guidelines (Learned from Review Passes)

### Comments

- **Do not write comments that restate the code.** `// Set as active provider` above `cfg.ActiveProvider = provider` adds noise. Comments should explain *why*, not *what*.
- **Do not leave "reserved for future use" or "temporary Phase X shim" comments.** They rot. If a parameter or code path is unused, either use it or remove it. Do not annotate it with speculative future plans.
- **Do not silence unused imports with `var _ = pkg.Symbol`.** Remove the import. If the import is genuinely needed later, add it back then.
- **Do not write section-header inline comments** (`// Help line`, `// Model list`, `// Cache the result`) that label the next few lines. The code is the label.
- **Keep godoc comments** on exported identifiers. These are documentation, not slop. `"GenerateRequest is the normalized request for image generation."` is correct and useful.
- **Comments that explain algorithm decisions are valuable.** `// If any channel is non-zero, treat as white` documents the binarization rule, not the code.
- **Flag-group separators** in `init()` blocks (`// Mask flags`, `// Output flags`) are acceptable as visual navigation in long registration blocks.

### Dead Code

- Before adding a parameter to a function, verify it will be used. If verification moves to a different layer (e.g. CLI does the `--force` check, not `auth.Add`), do not pass `force` through to the function that ignores it.
- Helper functions extracted "for future use" that are never called are dead code. Delete them. `friendlyPath()` survived multiple commits without a single caller.
- Always grep the full project before removing or renaming. A function may have callers in `*_test.go`, TUI files, or sibling packages that compile checks will flag only after staging.

### Input Validation Patterns

- **URL normalization:** Always `strings.TrimRight(baseURL, "/")` before URL joins. A trailing slash in `--base-url` or `POTACO_BASE_URL` produces double-slash endpoints (`/v1//images/generations`) that fail silently.
- **Image file bounds:** Every code path that reads and decodes a user-supplied image file must check `maxImageFileBytes` (file size) and `validateImageDimensionsFromBytes` (pixel count via header) *before* `image.Decode`. This includes `ReadImage`, `LoadMaskFile`, and any future function that accepts image paths. Without bounds, a large image can OOM the CLI.
- **Stdout with multiple images:** Never write multiple image blobs (PNG/JPEG) to `os.Stdout` in sequence. Downstream tools cannot decode the result. Reject `--stdout` with `--n > 1` early with a `UserError` and a helpful hint.
- **Cancellation propagation:** When a TUI picker returns `("", nil)` on cancel, the caller must check for the empty string and return early. Do not let an empty provider name flow through to `SetActiveProvider("")` which produces a confusing downstream error.

### Credential Lookup

- `GetActiveAPIKey()` returns the key for the *active* provider. When a specific provider name is given (e.g. `--provider fal` while `openai` is active), use `GetAPIKey(providerName)` instead. Never fall through to `GetActiveAPIKey()` when an explicit provider was specified.
- When adding a method to `AuthManager`, keep it thin: delegate to `m.store.Get(provider)` rather than reimplementing the lookup.

### Documentation In sync

- When adding a new command (`version`, `update`, `uninstall`), update all three docs in the same commit: `README.md` (user-facing), `AGENTS.md` (agent guidelines), `CONTRIBUTING.md` (contributor guide). Out-of-sync docs cause agents and contributors to miss features.
- When adding a CLI flag, add it to the README flag table in the same commit. Do not document flags that do not exist (e.g. `--view` was listed in the README with no corresponding code).
- The file structure tree in AGENTS.md and CONTRIBUTING.md must list every `.go` file. Missing files cause agents to not know they exist.

## Exit Codes

Defined in `internal/cli/errors.go`: 0 success, 2 config error, 3 API error, 4 image error. New code should use `configUserErr`, `apiUserErr`, `imageUserErr` constructors from `usererr.go` which produce `UserError` values with a friendly message and hint. Legacy `configError()`, `apiError()`, `imageError()` wrappers still exist for backward compatibility. `Execute()` in `root.go` renders the error (colored on TTY, plain text otherwise) and writes the raw error to `~/.potaco/debug.log` before exiting with the appropriate code.
