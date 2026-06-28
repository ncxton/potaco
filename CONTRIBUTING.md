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
./potaco auth add openrouter --type openai-compatible --api-key sk-... --base-url https://openrouter.ai/api/v1  # Connect named custom endpoint
./potaco auth list                          # List connected providers
./potaco use openai                         # Switch active provider
./potaco models                             # Pick a model interactively
./potaco models list                        # List available models without changing selection
./potaco status                             # Show current provider/model status
./potaco config set model gpt-image-2       # Set active provider model
./potaco config set model.edit true         # Mark active provider model edit capable
./potaco config set auto_update false       # Disable automatic update prompts
```

## Coding Style

- Go 1.26, pure Go only (no CGO).
- Run `gofmt -l .` before committing. Fix any flagged files.
- Run `go vet ./...` before committing. Fix any issues.
- No panics in library code. Use `fmt.Errorf` with `%w` for error wrapping.
- Keep files focused: one responsibility per file, one subcommand per file in `internal/cli/`.
- Table-driven tests are preferred. Test files sit alongside source: `foo.go` / `foo_test.go`.

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

Releases are automated via GoReleaser. To cut a new release, a maintainer creates a tag following semantic versioning (for example, `v1.0.0`) and pushes it. The release workflow then builds for linux/amd64, linux/arm64, darwin/amd64, and darwin/arm64, and creates a GitHub release with archives, checksums, and an auto-generated changelog.

Please coordinate with other maintainers before pushing a release tag.
