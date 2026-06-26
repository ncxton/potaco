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
  observability/         Request ID propagation, metrics collection, structured error context
    metrics.go           Metrics tracking, request ID context, structured error logging
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
scripts/                 Git hooks and installation scripts
  pre-commit.sh          Pre-commit: gofmt, go vet, go mod tidy
  pre-push.sh            Pre-push: go test ./...
  install-hooks.sh       Install git hooks into .git/hooks/
Makefile                 Build, test, coverage, staticcheck, complexity targets
.env.example             Environment variable template
.github/
  workflows/
    ci.yml               CI: build, vet, gofmt, staticcheck, gocyclo, coverage, go mod tidy
    security.yml         Gitleaks secret scanning, AGENTS.md validation
    release.yml          GoReleaser release automation
  CODEOWNERS             Code ownership assignments
  dependabot.yml         Dependency update automation (weekly)
  ISSUE_TEMPLATE/        Structured bug report and feature request forms
  PULL_REQUEST_TEMPLATE.md  PR template with checklist
.gitleaks.toml            Gitleaks secret scanning config
```

Layered monolith dependency graph:

```
cli --> adapter, auth, config, credential, tui, image, observability
tui  --> adapter, auth
auth --> config, credential
adapter/openai|fal|vercel --> adapter (parent), config, observability
config, credential, image, observability --> (no internal deps)
```

All packages live under `internal/` (not importable externally). Adapter sub-packages register themselves via `init()` and are imported for side effects in `cli/helpers.go`.

## Build, Test, and Development Commands

```
go build -o potaco .       # Build the binary
go test ./...              # Run all tests
go test ./... -v           # Run all tests with verbose output
go test ./... -coverprofile=coverage.out -covermode=atomic  # Run tests with coverage
go tool cover -func=coverage.out  # Show coverage summary
go vet ./...               # Lint
gofmt -l .                 # Check formatting (should output nothing)
make build                 # Build via Makefile
make test                  # Test via Makefile
make cover                 # Coverage report via Makefile
make staticcheck           # Run staticcheck (dead code, complexity, unused)
make complexity            # Run gocyclo (cyclomatic complexity, threshold 30)
make tidy                  # Run go mod tidy
make check                 # Run vet, fmt, test
```

**Pre-commit hooks**: Install locally after cloning:
```
sh scripts/install-hooks.sh   # Installs pre-commit (gofmt, vet, tidy) and pre-push (tests) hooks
```

**Static analysis tools configured in CI**:
- `go vet ./...` - Standard Go linter
- `gofmt -l .` - Format check
- `staticcheck` - Dead code, complexity, unused variable detection
- `gocyclo -over 30 .` - Cyclomatic complexity threshold enforcement
- `go mod tidy` - Unused dependency detection (fails if go.mod/go.sum not tidy)
- `go test -coverprofile` - Coverage measurement with artifact upload
- `gitleaks` - Secret scanning on all PRs and scheduled

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

## Code Quality Guidelines

### Comments

- Comments should explain *why*, not *what*. If the code is self-explanatory, do not add a comment.
- Do not leave "reserved for future use," "temporary shim," or speculative phase comments. They rot. Unused code or parameters should be removed, not annotated.
- Do not silence unused imports with `var _ = pkg.Symbol`. Remove the import. Add it back when it is actually needed.
- Do not write inline section-header comments that label the next few lines. The code is the label.
- Keep godoc comments on exported identifiers. These are documentation, not slop.
- Comments that explain algorithm decisions or non-obvious logic are valuable. Keep them.

### Dead Code

- Do not add parameters to a function that the function body ignores. If a concern moves to a different layer, update the signature.
- Helper functions extracted "for future use" that are never called are dead code. Delete them.
- Grep the full project (including test files) before removing or renaming. Callers may exist in sibling packages or tests.

### Input Validation

- Normalize user-supplied URLs (trim trailing slashes) before joining with path segments to avoid double-slash endpoints.
- Validate file size and dimensions before decoding user-supplied image files to prevent OOM. Use shared budget checks (`maxImageFileBytes`, `validateImageDimensionsFromBytes`) for every image input path.
- Do not write multiple binary blobs to stdout in sequence. Reject combinations of flags that would produce undecodable output early with a `UserError`.
- When an interactive picker returns a zero-value result on cancel, the caller must check for it and return early. Do not let a cancelled selection flow through to downstream operations.

### Credential and Provider Lookup

- When a specific provider is requested by name, look up credentials for that provider. Do not fall through to a "get active" lookup when an explicit provider was given.
- Keep `AuthManager` methods thin: delegate to the credential store rather than reimplementing logic.

### Documentation Sync

- When adding a new command or flag, update `README.md`, `AGENTS.md`, and `CONTRIBUTING.md` in the same commit.
- Do not document flags, features, or commands that do not exist in the code.
- Keep file structure trees current. Missing files cause agents and contributors to miss code that exists.

## Exit Codes

Defined in `internal/cli/errors.go`: 0 success, 2 config error, 3 API error, 4 image error. New code should use `configUserErr`, `apiUserErr`, `imageUserErr` constructors from `usererr.go` which produce `UserError` values with a friendly message and hint. Legacy `configError()`, `apiError()`, `imageError()` wrappers still exist for backward compatibility. `Execute()` in `root.go` renders the error (colored on TTY, plain text otherwise) and writes the raw error to `~/.potaco/debug.log` before exiting with the appropriate code.
