# Repository Guidelines

Potaco is a Go CLI for image generation and editing via OpenAI-compatible providers.

## Project Structure & Module Organization

```
main.go              Entry point, calls cli.Execute()
internal/
  cli/               Cobra commands (root, gen, edit, config, info), helpers, output, errors
  config/            Config file loading, env var parsing, merge precedence logic
  provider/          HTTP client for /v1/images/generations and /v1/images/edits, retries, presets
  image/             Image I/O, mask generation, outpaint canvas, terminal display
docs/superpowers/    Design spec and implementation plan
```

Layered monolith: `cli` depends on `provider`, `config`, and `image`. `provider` depends on `config` types. `config` and `image` have no internal dependencies. All packages live under `internal/` (not importable externally).

## Build, Test, and Development Commands

```
go build -o potaco .       # Build the binary
go test ./...              # Run all tests
go test ./... -v           # Run all tests with verbose output
go vet ./...               # Lint
gofmt -l .                 |# Check formatting (should output nothing)
```

Config file lives at `~/.potaco/config.yaml`. For local testing without a real provider, use `--dry-run`:
```
POTACO_BASE_URL=https://api.openai.com POTACO_API_KEY=sk-test \
  ./potaco gen --prompt "a cat" --dry-run
```

## Coding Style & Naming Conventions

- Go 1.26, pure Go only (no CGO).
- Standard `gofmt` formatting. Run `gofmt -l .` before committing; fix any flagged files.
- Internal image package is imported as `img` in CLI files to avoid collision with stdlib `image`.
- No panics in library code. Use `fmt.Errorf` with `%w` for error wrapping.
- Keep files focused: one responsibility per file, one subcommand per file in `cli/`.
- Table-driven tests are preferred. Test files sit alongside source: `foo.go` / `foo_test.go`.

## Testing Guidelines

- Testing framework: Go standard `testing` package with `httptest.Server` for provider tests.
- TDD: write failing tests first, implement to pass, then commit.
- CLI tests dispatch via `rootCmd.SetArgs([]string{"subcommand", ...})` and `rootCmd.Execute()` (not subcommand-direct `Execute()`).
- Provider tests use `httptest.Server` mocks and override `client.backoff` to 1ms for fast retry tests.
- Image tests use `t.TempDir()` for temp files and `bytes.Buffer` for in-memory roundtrips.

## Commit & Pull Request Guidelines

Conventional commit style scoped by package:
```
config: add merge logic with flag>env>file>default precedence
provider: add client with Generate method and response parsing
image: add mask generation (rect, circle, file loading with scaling)
cli: add gen subcommand, shared helpers, and dry-run support
fix: retry body reset, config file perms 0600, provider preset in merge, exit codes
```

Subject line is lowercase, no period, prefixed with package name or `fix:`. Reference the design spec (`docs/superpowers/specs/`) and plan (`docs/superpowers/plans/`) when adding features.

## Exit Codes

Defined in `internal/cli/errors.go`: 0 success, 2 config error, 3 API error, 4 image error. Wrap errors with `configError()`, `apiError()`, or `imageError()` so `Execute()` emits the correct exit code.
