# Contributing to Potaco

Thanks for your interest in contributing to Potaco! This guide covers development setup, coding standards, and the pull request process.

## Development Setup

```sh
git clone https://github.com/ncxton/potaco.git
cd potaco
go build -o potaco .
go test ./...
```

For local testing without a real provider:

```sh
POTACO_BASE_URL=https://api.openai.com POTACO_API_KEY=sk-test \
  ./potaco gen --prompt "a cat" --dry-run
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
main.go              Entry point, calls cli.Execute()
internal/
  cli/               Cobra commands (root, gen, edit, edit_mask, config, info), helpers, output, errors
  config/            Config file loading, env var parsing, merge precedence logic
  provider/          HTTP client for /v1/images/generations and /v1/images/edits, retries, presets
  image/             Image I/O, mask generation, outpaint canvas, terminal display
```

## Commit Style

Conventional commits scoped by package:

```
config: add merge logic with flag>env>file>default precedence
provider: add client with Generate method and response parsing
cli: add gen subcommand, shared helpers, and dry-run support
fix: retry body reset, config file perms 0600
```

Subject line is lowercase, no period, prefixed with package name or `fix:`.

## Pull Request Process

1. Create a branch from `main`
2. Run `go test ./...` and `go vet ./...`
3. Ensure `gofmt -l .` outputs nothing
4. Open a PR with a conventional commit-style title
5. CI runs automatically: build, vet, gofmt, test

## Testing Guidelines

- CLI tests dispatch via `rootCmd.SetArgs([]string{"subcommand", ...})` and `rootCmd.Execute()`
- Provider tests use `httptest.Server` mocks and override `client.backoff` to 1ms for fast retry tests
- Image tests use `t.TempDir()` for temp files and `bytes.Buffer` for in-memory roundtrips
- Use `--dry-run` for local testing without a real provider

## Releasing (Maintainers)

Releases are automated via GoReleaser. To cut a new release:

```sh
git tag v1.0.0
git push --tags
```

This triggers the release workflow which builds for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, creates GitHub release with archives, checksums, and auto-generated changelog.
