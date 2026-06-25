# Repository Guidelines

Potaco is a Go CLI for image generation and editing via OpenAI-compatible providers.

## Mandatory Skill Loading (Before Any Codebase Action)

Before taking **any** action on the codebase (reading, writing, editing, refactoring, or reviewing Go source), the following skills **must** be loaded via the `Skill` tool. On-demand reference loading within each skill is required when the skill instructs it.

### `programming` (Go focus)

- **When**: before writing or modifying any `.go` file, `go.mod`, `go.sum`, `.golangci.yml`, or `*.proto` next to the Go module.
- **How**: invoke `Skill("programming")`, then read `references/go/README.md` as the skill's Phase 0 language gate mandates. Load additional `references/go/*` files on demand per the skill's Go jump table (e.g., `libraries.md`, `golangci-strict.md`, `type-patterns.md`, `error-handling.md`, `concurrency.md`, `cobra-stack.md`, `testing.md`).
- **Why**: enforces the shared philosophy (type-strict, parse-don't-validate, branded primitives, exhaustive matching, no bare error strings, RAII resources, boundary-only catch) and the Go-specific iron list (no `interface{}`/bare `any` in domain sigs, no `_ = err`, `defer x.Close()` immediately, `context.Context` as first param, `errgroup` for structured concurrency, `-race` on every test). The skill also mandates the post-write review loop (LOC ceiling, architectural self-review) and the TDD discipline (red, green, refactor).

### `remove-ai-slops`

- **When**: after any code-writing session before declaring work complete, or when the branch contains AI-authored patterns (broad `except`/empty `catch`, redundant null checks, vague TODOs, oversized modules, dead helpers, redundant post-action verification), or when the user says "remove slop", "clean AI code", "deslop", "clean up AI-generated code".
- **How**: invoke `Skill("remove-ai-slops")`, then follow its six-phase process (scope, lock behavior with regression tests, cleanup plan, parallel removal in batches of 5, quality gates, critical review).
- **Why**: enforces the safety invariant that behavior is locked by green tests before any line is removed, and the ten slop categories (obvious comments, over-defensive code, excessive complexity, needless abstraction, boundary violations, dead code, duplication, performance equivalences, missing tests, oversized modules >250 pure LOC). The 250-pure-LOC ceiling is a defect, not a style preference.

### Order of operations

1. Load `programming` (Go) first, before writing or editing any Go source.
2. After the code-writing session, before declaring done, load `remove-ai-slops` and run the post-write review loop. If any code smell fires (250+ LOC, >3 params, redundant verification, negative naming), or the post-write loop surfaces 2+ issues, the `refactor` skill may also be needed per the `programming` skill's companion-skills table.

No exceptions for "small" or "one-off" code. Production hygiene applies to disposable scripts (`//go:build ignore` + `go run`) as well.

## Project Structure & Module Organization

```
main.go              Entry point, calls cli.Execute()
internal/
  cli/               Cobra commands (root, gen, edit, edit_mask, config, info), helpers, output, errors
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
