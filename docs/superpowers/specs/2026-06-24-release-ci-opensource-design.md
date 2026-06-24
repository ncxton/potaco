# Release Distribution, PR Validation, and Open-Source Setup

**Date:** 2026-06-24  
**Status:** Approved  
**Scope:** GitHub Actions CI/release workflows, native installer script, open-source files, module path fix

---

## Context

Potaco is a Go CLI for image generation and editing via OpenAI-compatible providers. The project is MVP-complete with gen/edit/config/info subcommands, all tests passing, and no CI, no release pipeline, no README, no LICENSE. The go.mod module path is `github.com/ngct/potaco` but the actual GitHub account is `ncxton`. The project needs to be prepared for open-source publication on GitHub.

## Goals

1. Fix the GitHub username across the codebase (`ngct` -> `ncxton`)
2. Add a PR validation GitHub Actions workflow (build, test, vet, gofmt)
3. Add a release distribution pipeline using GoReleaser, triggered by git tag pushes
4. Create a native installer script with a beautifully designed TUI (interactive by default, `POTACO_NON_INTERACTIVE=1` for CI/automation)
5. Create open-source files: README.md, LICENSE (MIT), CONTRIBUTING.md
6. Add the git remote pointing to `github.com/ncxton/potaco`

## Non-Goals

- Homebrew tap formula (future, after first release)
- Windows support (MVP phase is macOS + Linux only)
- Snapshot/nightly builds (not needed for MVP)
- Container image publishing (not needed for MVP)

---

## 1. GitHub Username Fix and Remote Setup

### Problem

The go.mod module path is `github.com/ngct/potaco` but the verified GitHub account is `ncxton` (confirmed via `gh api user --jq .login`).

### Changes

- **`go.mod`**: `module github.com/ngct/potaco` -> `module github.com/ncxton/potaco`
- **All `.go` source files**: update import paths from `github.com/ngct/potaco/internal/...` to `github.com/ncxton/potaco/internal/...`
  - `main.go` (1 import)
  - `internal/cli/edit.go` (2 imports: config, provider)
  - `internal/cli/config_cmd.go` (2 imports: config, provider)
  - `internal/cli/gen.go` (2 imports: config, provider)
  - `internal/cli/info.go` (1 import: image)
  - `internal/cli/edit_mask.go` (1 import: image)
  - `internal/cli/config_cmd_test.go` (1 import: config)
  - `internal/cli/helpers.go` (3 imports: config, image, provider)
- **`go mod tidy`**: run after the rename to ensure go.sum is consistent
- **Git remote**: `git remote add origin https://github.com/ncxton/potaco.git`
- **Historical docs**: `docs/superpowers/specs/2026-06-24-potaco-cli-design.md` and `docs/superpowers/plans/2026-06-24-potaco-cli.md` retain the old `ngct` path as historical records. These are not source code and do not affect the build.

### Verification

- `go build ./...` passes
- `go test ./...` passes
- `go vet ./...` passes
- `gofmt -l .` outputs nothing

---

## 2. PR Validation Workflow

**File:** `.github/workflows/ci.yml`

### Triggers

- `pull_request` targeting `main`
- `push` to `main`

### Job: `validate`

- **Runner:** `ubuntu-latest`
- **Go version:** 1.26 (via `actions/setup-go@v5`)
- **Steps:**
  1. `actions/checkout@v4`
  2. `actions/setup-go@v5` with `go-version: '1.26'` (includes automatic module caching via go.sum)
  3. `go mod download`
  4. `go build ./...` (compilation check)
  5. `go vet ./...` (static analysis)
  6. `gofmt -l .` check: `if [ -n "$(gofmt -l .)" ]; then echo "::error::files need gofmt"; gofmt -l .; exit 1; fi`
  7. `go test ./... -v` (all tests with verbose output)

### Concurrency

Cancel in-progress runs for the same PR/branch when a new push arrives:
```yaml
concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true
```

---

## 3. Release Workflow

**File:** `.github/workflows/release.yml`

### Triggers

- `push` with tags matching `v*` (e.g., `v1.0.0`, `v0.1.0-beta`)

### Job: `release`

- **Runner:** `ubuntu-latest`
- **Permissions:** `contents: write` (for creating GitHub releases)
- **Steps:**
  1. `actions/checkout@v4` with `fetch-depth: 0` (GoReleaser needs full git history for changelog)
  2. `actions/setup-go@v5` with `go-version: '1.26'`
  3. Install GoReleaser: `go install github.com/goreleaser/goreleaser@latest` (or via a versioned action)
  4. Run: `goreleaser release --clean --fail-fast`

### GoReleaser Configuration

**File:** `.goreleaser.yaml`

```yaml
project_name: potaco

before:
  hooks:
    - go mod download

builds:
  - id: potaco
    main: .
    binary: potaco
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64

archives:
  - format: tar.gz
    name_template: >-
      {{ .ProjectName }}_{{ .Version }}_
      {{- if eq .Os "darwin" }}apple-darwin
      {{- else }}{{ .Os }}
      {{- end }}_{{ .Arch }}
    files:
      - install.sh
      - LICENSE
      - README.md

checksum:
  name_template: '{{ .ProjectName }}_{{ .Version }}_checksums.txt'
  algorithm: sha256

changelog:
  use: git
  sort: asc
  groups:
    - title: Features
      regexp: '^feat:'
      order: 0
    - title: Bug Fixes
      regexp: '^fix:'
      order: 1
    - title: CLI
      regexp: '^cli:'
      order: 2
    - title: Other
      order: 999
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'

release:
  github:
    owner: ncxton
    name: potaco
  draft: false
  prerelease: auto
  name_template: '{{ .Tag }}'
```

### Archive Naming

Archives use a clear naming convention for the installer script's download URL:
- `potaco_1.0.0_linux_amd64.tar.gz`
- `potaco_1.0.0_linux_arm64.tar.gz`
- `potaco_1.0.0_apple-darwin_amd64.tar.gz`
- `potaco_1.0.0_apple-darwin_arm64.tar.gz`

Each archive contains: the `potaco` binary, `install.sh`, `LICENSE`, `README.md`.

---

## 4. Installer Script

**File:** `install.sh` (repo root, also included in release archives)

### Overview

A POSIX-compliant shell script that downloads the correct potaco binary for the user's platform from GitHub Releases, verifies the checksum, and installs it. Interactive by default with a clean TUI. `POTACO_NON_INTERACTIVE=1` enables fully silent mode for CI/automation.

### Platform Detection

- OS: `uname -s` maps `Darwin` -> `apple-darwin`, `Linux` -> `linux`. Other OSes are rejected with a message listing supported platforms.
- Arch: `uname -m` maps `x86_64`/`amd64` -> `amd64`, `aarch64`/`arm64` -> `arm64`. Other arches are rejected.

### Version Detection

- Query `https://api.github.com/repos/ncxton/potaco/releases/latest` using `curl` or `wget` (whichever is available)
- Parse `tag_name` from the JSON response using `grep`/`sed` (no dependency on `jq`)
- If the API call succeeds: set `VERSION` to the parsed tag (e.g., `v1.0.0`) and use `https://github.com/ncxton/potaco/releases/download/${VERSION}/...` for downloads
- If the API call fails: set `VERSION` to empty and fall back to GitHub's `latest` redirect URLs: `https://github.com/ncxton/potaco/releases/latest/download/potaco_${OS}_${ARCH}.tar.gz` and `https://github.com/ncxton/potaco/releases/latest/download/potaco_checksums.txt`. GitHub automatically redirects these to the actual latest release assets. In this fallback mode, the progress text shows " Installing latest..." since the version is unknown.

### Download and Verification

- If `VERSION` is known (API succeeded):
  - Download URL: `https://github.com/ncxton/potaco/releases/download/${VERSION}/potaco_${VERSION}_${OS}_${ARCH}.tar.gz`
  - Checksums file URL: `https://github.com/ncxton/potaco/releases/download/${VERSION}/potaco_${VERSION}_checksums.txt`
- If `VERSION` is unknown (API failed, fallback mode):
  - Download URL: `https://github.com/ncxton/potaco/releases/latest/download/potaco_${OS}_${ARCH}.tar.gz`
  - Checksums file URL: `https://github.com/ncxton/potaco/releases/latest/download/potaco_checksums.txt`
- Verify: compute SHA-256 of the downloaded tarball and compare against the checksums file
  - Use `sha256sum` if available, otherwise `shasum -a 256`
  - If neither is available: print a warning and skip verification (do not fail the install, since some minimal containers lack these tools)
- Download retry: up to 3 attempts with 2-second backoff between retries

### Installation

- Default target: `/usr/local/bin/potaco`
  - If `/usr/local/bin` is writable without sudo: install directly
  - If sudo is available and interactive mode: prompt "Potaco needs sudo to install to /usr/local/bin. Continue? [Y/n]"
  - If sudo is not available or not interactive: fallback to `~/.local/bin/potaco`
- Fallback target: `~/.local/bin/potaco`
  - Create `~/.local/bin` if it does not exist
  - If `~/.local/bin` is not in `$PATH`: print a warning with instructions to add it
- After install: `chmod +x` the binary

### Interactive TUI (Default Mode)

**Aesthetic: modern clean, single accent color (cyan/ANSI 36)**

1. **Banner**: ASCII art potaco logo (~8 lines tall) inside a `+-` bordered box, tagline below
2. **Platform info**: detected OS and arch in a small info box
3. **Version**: "Installing potaco v1.0.0..." with the resolved version number
3. **Progress spinner**: 8-frame ASCII spinner (`|`, `/`, `-`, `\` cycling via `\r` redraw) with current step text:
   - "Downloading..."
   - "Verifying checksum..."
   - "Extracting..."
   - "Installing..."
4. **Success box**: green-bordered box with:
   - "Potaco installed successfully!"
   - Install path
   - Next steps: `potaco config set --base-url <url> --api-key <key>` and `potaco gen --prompt "hello"`
   - Docs link: `https://github.com/ncxton/potaco#readme`
5. **Errors**: red-bordered box with error message and remediation hint

**Color handling:**
- Cyan (ANSI 36) for borders, headers, key info
- Green (ANSI 32) for success
- Red (ANSI 31) for errors
- Yellow (ANSI 33) for warnings
- Respect `NO_COLOR` env var: strip all ANSI codes
- Respect `TERM=dumb`: strip spinner animation, keep text output

### Non-Interactive Mode (`POTACO_NON_INTERACTIVE=1`)

- No ASCII art, no spinner, no colored boxes
- Plain text lines: "Detected: linux/amd64", "Downloading potaco v1.0.0...", "Installing to /usr/local/bin/potaco...", "Done."
- No prompts: use `/usr/local/bin` if writable without sudo, otherwise `~/.local/bin` without asking
- Exit 0 on success, non-zero on failure
- Suitable for: `curl -fsSL https://github.com/ncxton/potaco/releases/latest/download/install.sh | POTACO_NON_INTERACTIVE=1 sh`

### Error Handling

- Missing `curl` and `wget`: error with hint to install one of them
- Missing `tar`: error with hint
- Missing `sha256sum` and `shasum`: warning (skip verification, continue install)
- Network/HTTP failure: retry up to 3 times, then error with clear message
- Checksum mismatch: error, do not install, do not leave any artifacts
- Unsupported OS/arch: error with list of supported platforms

### Script Structure

```
#!/bin/sh
set -eu

# --- Helpers ---
info()      # cyan text
warn()      # yellow text
error()     # red text, to stderr
success()   # green text
print_banner()      # ASCII art in a box
print_box()         # generic bordered box with title + content
spinner_start()     # start background spinner with message
spinner_stop()       # stop spinner, clear line

# --- Platform detection ---
detect_platform()    # returns OS and ARCH, or exits with error

# --- Version detection ---
detect_version()     # returns latest version tag from GitHub API

# --- Download and verify ---
download_and_verify() # downloads tarball + checksums, verifies, returns temp path

# --- Install ---
install_binary()     # moves binary to target dir, chmod +x

# --- Main ---
main()               # orchestrates the flow, checks NON_INTERACTIVE
```

**Estimated size:** ~400-500 lines, single file, no external dependencies beyond standard POSIX utilities (`sh`, `uname`, `curl`/`wget`, `tar`, `shasum`/`sha256sum`).

---

## 5. Open-Source Files

### README.md

**Structure:**

1. **Hero banner**: ASCII art potaco logo + tagline "Terminal image generation and editing CLI"
2. **Badges**: CI status, Go Report Card, license MIT, latest release
3. **Features**: bullet list of key capabilities
4. **Installation**: 
   - One-liner (curl|sh with interactive installer)
   - Non-interactive one-liner (for CI)
   - Manual download from GitHub Releases
5. **Quick start**: `potaco config set` then `potaco gen`
6. **Usage** (one section per subcommand):
   - `potaco gen` with flags table and examples
   - `potaco edit` with flags table and examples
   - `potaco config` with subcommands and examples
   - `potaco info` with examples
7. **Configuration**: config.yaml structure, env vars, precedence
8. **Provider presets**: table of built-in presets
9. **Contributing**: link to CONTRIBUTING.md
10. **License**: MIT

### LICENSE

Standard MIT license text.
- Copyright: 2026 ncxton
- Full MIT text as published by OSI

### CONTRIBUTING.md

1. **Development setup**: clone, build, test
2. **Coding style**: gofmt, go vet, no panics, error wrapping, table-driven tests
3. **Project structure**: brief description of internal/ packages
4. **Commit style**: conventional commits scoped by package, with examples
5. **Pull request process**: branch, test, vet, gofmt, open PR
6. **Testing**: dry-run mode, httptest pattern, t.TempDir pattern
7. **Releasing**: tag push triggers GoReleaser (for maintainers)

---

## 6. File Inventory

New files to create:
- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `.goreleaser.yaml`
- `install.sh`
- `README.md`
- `LICENSE`
- `CONTRIBUTING.md`

Files to modify:
- `go.mod` (module path)
- `main.go` (import path)
- `internal/cli/edit.go` (import paths)
- `internal/cli/config_cmd.go` (import paths)
- `internal/cli/gen.go` (import paths)
- `internal/cli/info.go` (import path)
- `internal/cli/edit_mask.go` (import path)
- `internal/cli/config_cmd_test.go` (import path)
- `internal/cli/helpers.go` (import paths)

Git operations:
- `git remote add origin https://github.com/ncxton/potaco.git`

---

## 7. Testing and Verification

### Build verification
- `go build ./...` passes after module path change
- `go test ./...` passes
- `go vet ./...` passes
- `gofmt -l .` outputs nothing

### CI workflow verification
- The `ci.yml` workflow YAML is valid
- All steps reference correct Go version and commands
- Concurrency group prevents redundant runs

### Release workflow verification
- `.goreleaser.yaml` is valid: `goreleaser check` (if GoReleaser is installed)
- The release workflow triggers on `v*` tags
- GoReleaser builds for all 4 platform targets (linux/darwin x amd64/arm64)
- Archives include install.sh, LICENSE, README.md alongside the binary

### Installer script verification
- `install.sh` is POSIX-compliant: `sh -n install.sh` (syntax check)
- Platform detection handles Darwin/Linux, amd64/arm64 correctly
- Non-interactive mode produces no prompts
- Checksum verification works when sha256sum/shasum is present
- Graceful degradation when checksum tools are missing
