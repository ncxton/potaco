# Repository Guidelines

Potaco is a Go CLI for image generation and editing via multi-provider adapters (OpenAI, fal, Vercel AI Gateway, and custom OpenAI-compatible endpoints) with encrypted credential storage and interactive TUI flows.

## Project Structure

Provider presets in `internal/cli/helpers.go` store only a base URL (`BaseURL`).
There is no `DefaultModel` preset; models are selected by the user via `potaco
models` or `potaco config set model <model>`. The `custom` provider has no preset and
requires a user-supplied base URL.

Layered monolith dependency graph:

```
cli --> adapter, auth, config, credential, tui, image
tui  --> adapter, auth
auth --> config, credential
adapter/openai|fal|vercel|custom --> adapter (parent), config
config, credential, image --> (no internal deps)
```

All packages live under `internal/` (not importable externally). Adapter sub-packages, including `custom`, register themselves via `init()` and are imported for side effects in `cli/helpers.go`.

## Build, Test, and Development Commands

```
go build -o potaco .       # Build the binary
make setup                  # Install pre-commit hooks (gofmt, vet, tidy) and pre-push (tests)
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
make duplicates            # Run jscpd (duplicate code detection, threshold 5%)
make tech-debt             # Scan for TODO/FIXME without issue references
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
- `jscpd` - Duplicate code detection (threshold 5%, min 20 lines/100 tokens)
- `grep` - Tech debt marker scanner (TODO/FIXME must reference issues)
- `go mod tidy` - Unused dependency detection (fails if go.mod/go.sum not tidy)
- `go test -coverprofile` - Coverage measurement with artifact upload
- `gitleaks` - Secret scanning on all PRs and scheduled

Config file lives at `~/.potaco/config.yaml`. Update check cache lives at `~/.potaco/.potaco.json`. Encrypted credentials at `~/.potaco/credentials.enc` with salt at `~/.potaco/.salt`. For local testing without a real provider, use `--dry-run`:
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
