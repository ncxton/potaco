# Contributing to Potaco

Thanks for your interest in contributing to Potaco! This guide covers development setup, coding standards, and the pull request process.

## Development Setup

```sh
git clone https://github.com/ncxton/potaco.git
cd potaco
go build -o potaco .
go test ./...
```

### Pre-commit Hooks

Install git hooks for local quality checks:

```sh
sh scripts/install-hooks.sh
```

This installs:
- **pre-commit**: runs `gofmt`, `go vet`, and `go mod tidy` check
- **pre-push**: runs `go test ./...`

### Static Analysis

CI runs the following static analysis tools. Install locally for faster feedback:

```sh
go install honnef.co/go/tools/cmd/staticcheck@latest   # Dead code, complexity
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest  # Cyclomatic complexity
make staticcheck   # Run staticcheck
make complexity    # Run gocyclo (threshold: 30)
make duplicates    # Run jscpd (duplicate code, threshold: 5%)
make tech-debt     # Scan for TODO/FIXME without issue references
make cover         # Generate coverage report
```

For local testing without a real provider:

```sh
POTACO_BASE_URL=https://api.openai.com POTACO_API_KEY=sk-test \
  ./potaco gen --prompt "a cat" --dry-run
```

Provider credentials are managed via `auth add`:

```sh
./potaco auth add openai --api-key sk-...   # Connect a provider
./potaco auth add custom --api-key sk-... --base-url https://api.example.com/v1  # Connect custom endpoint
./potaco auth list                          # List connected providers
./potaco use openai                         # Switch active provider
./potaco models                             # Pick a model interactively
./potaco models list                        # List available models without changing selection
./potaco status                             # Show current provider/model status
```

## Coding Style

- Go 1.26, pure Go only (no CGO).
- Run `gofmt -l .` before committing. Fix any flagged files.
- Run `go vet ./...` before committing. Fix any issues.
- No panics in library code. Use `fmt.Errorf` with `%w` for error wrapping.
- Keep files focused: one responsibility per file, one subcommand per file in `internal/cli/`.
- Table-driven tests are preferred. Test files sit alongside source: `foo.go` / `foo_test.go`.

## Project Structure

```
main.go                  Entry point, calls cli.Execute()
internal/
  cli/                   Cobra commands and CLI infrastructure
    root.go              Root command, persistent flags (--json, --verbose, --non-interactive)
    gen.go               gen subcommand (text-to-image)
    edit.go, edit_mask.go  edit subcommand (image editing, inpainting, outpainting)
    auth_cmd.go          auth add/remove/list subcommands
    config_cmd.go        config set/show subcommands
    models_cmd.go        models subcommand (discover and pick models)
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
    adapter.go           Adapter interface (Generate, Edit, DiscoverModels, Verify, SupportsGenerate, SupportsEdit)
    registry.go          Factory registry: Register/Get/List for provider adapters
    openai/              OpenAI adapter (Images API, /v1/images/generations, /v1/images/edits)
    fal/                 fal adapter (fal.run inference, api.fal.ai discovery, image-to-image)
    vercel/              Vercel AI Gateway adapter (generate-only, no edit support)
    custom/              OpenAI-compatible custom provider adapter (user-supplied base URL)
      custom.go          Adapter struct, AuthHeader, Name, SupportsGenerate, SupportsEdit
      generate.go        Generate (text-to-image)
      edit.go            Edit (inpainting with mask)
      discover.go        DiscoverModels (GET /models), Verify
      response.go        Response types
      retry.go           Retry with exponential backoff
  auth/                  AuthManager: coordinates credential store and multi-provider config
  credential/            Encrypted credential storage (AES-256-GCM, machine-derived key)
  config/                Multi-provider YAML config (~/.potaco/config.yaml)
  observability/         Request ID propagation, metrics collection, structured error context
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
Makefile                 Build, test, coverage, staticcheck, complexity targets
.env.example             Environment variable template
```

All packages live under `internal/`. Adapter sub-packages register via `init()` and are blank-imported in `cli/helpers.go`. The `custom` adapter has no preset base URL; tests and users must supply one via `--base-url` or `base_url` in config.

## Commit Style

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

## Pull Request Process

1. Create a branch from `main`
2. Run `go test ./...` and `go vet ./...`
3. Ensure `gofmt -l .` outputs nothing
4. Run `go mod tidy` and commit any changes
5. Run `staticcheck ./...` and `gocyclo -over 30 .` locally
6. Open a PR with a conventional commit-style title
7. CI runs automatically: build, vet, gofmt, staticcheck, gocyclo, coverage, go mod tidy, gitleaks

### CI Checks

The following checks run on every PR:
- **Build**: `go build ./...`
- **Vet**: `go vet ./...`
- **Format**: `gofmt -l .`
- **Staticcheck**: Dead code, complexity, unused variable detection
- **Cyclomatic complexity**: `gocyclo -over 30 .`
- **Duplicate code**: `jscpd` (threshold 5%, min 20 lines/100 tokens)
- **Tech debt markers**: TODO/FIXME must reference issues (e.g., `TODO(#123)`)
- **go mod tidy**: Ensures no unused dependencies
- **Coverage**: `go test -coverprofile` with artifact upload
- **Secret scanning**: Gitleaks on all changes
- **Tests**: `go test ./... -v`
- **Cross-compile**: Builds for linux/darwin amd64/arm64
- **Static binary verification**: Ensures CGO_ENABLED=0

## Testing Guidelines

- CLI tests dispatch via `rootCmd.SetArgs([]string{"subcommand", ...})` and `rootCmd.Execute()`
- Adapter tests use `httptest.Server` mocks and override `Adapter.backoff` and `Adapter.sleep` (via `SetBackoff`/`SetSleep`) to 1ms for fast retry tests
- Credential tests verify encrypt/decrypt roundtrips with test keys and temp directories
- Image tests use `t.TempDir()` for temp files and `bytes.Buffer` for in-memory roundtrips
- TUI tests are minimal (smoke tests) since interactive forms require a TTY
- Use `--dry-run` for local testing without a real provider

## Releasing (Maintainers)

Releases are automated via GoReleaser. To cut a new release:

```sh
git tag v1.0.0
git push --tags
```

This triggers the release workflow which builds for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, creates GitHub release with archives, checksums, and auto-generated changelog.
