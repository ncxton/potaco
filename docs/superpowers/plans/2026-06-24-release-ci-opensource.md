# Release Distribution, PR Validation, and Open-Source Setup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the GitHub username across the codebase, add CI and release GitHub Actions workflows with GoReleaser, create a native installer script with a TUI, and add open-source files (README, LICENSE, CONTRIBUTING.md).

**Architecture:** Three independent tracks: (1) module path rename from `github.com/ngct/potaco` to `github.com/ncxton/potaco` with build verification, (2) CI/release workflow YAML files and GoReleaser config, (3) a POSIX-compliant `install.sh` installer and open-source documentation files.

**Tech Stack:** Go 1.26, GitHub Actions, GoReleaser v2 (goreleaser-action@v7), POSIX sh, ANSI escape codes.

## Global Constraints

- GitHub username is `ncxton` (verified via `gh api user --jq .login`)
- Go version is 1.26
- GoReleaser v2 requires `version: 2` in `.goreleaser.yaml`
- GoReleaser GitHub Action is `goreleaser/goreleaser-action@v7` with `version: "~> v2"`
- Platforms: macOS (darwin/amd64, darwin/arm64) and Linux (linux/amd64, linux/arm64) only
- Installer script is POSIX sh (`#!/bin/sh`), no bashisms
- Installer interactive mode is default; `POTACO_NON_INTERACTIVE=1` disables TUI
- Commit style: conventional commits scoped by package, lowercase, no period
- Config file: `~/.potaco/config.yaml`
- No CGO: `CGO_ENABLED=0` in GoReleaser builds

---

## File Structure

### New files

| File | Responsibility |
|------|----------------|
| `.github/workflows/ci.yml` | PR validation: build, vet, gofmt, test |
| `.github/workflows/release.yml` | Release: GoReleaser on tag push |
| `.goreleaser.yaml` | GoReleaser v2 config: build targets, archives, checksums, changelog |
| `install.sh` | POSIX installer: platform detect, download, verify, install, TUI |
| `README.md` | Project README with banner, install, usage, config |
| `LICENSE` | MIT license text |
| `CONTRIBUTING.md` | Contributing guide |

### Modified files

| File | Change |
|------|--------|
| `go.mod` | Module path `github.com/ngct/potaco` -> `github.com/ncxton/potaco` |
| `main.go` | Import path `github.com/ngct/potaco/internal/cli` -> `github.com/ncxton/potaco/internal/cli` |
| `internal/cli/edit.go` | 2 import paths |
| `internal/cli/config_cmd.go` | 2 import paths |
| `internal/cli/gen.go` | 2 import paths |
| `internal/cli/info.go` | 1 import path |
| `internal/cli/edit_mask.go` | 1 import path |
| `internal/cli/config_cmd_test.go` | 1 import path |
| `internal/cli/helpers.go` | 3 import paths |

---

### Task 1: Fix module path and all import paths

**Files:**
- Modify: `go.mod`
- Modify: `main.go`
- Modify: `internal/cli/edit.go`
- Modify: `internal/cli/config_cmd.go`
- Modify: `internal/cli/gen.go`
- Modify: `internal/cli/info.go`
- Modify: `internal/cli/edit_mask.go`
- Modify: `internal/cli/config_cmd_test.go`
- Modify: `internal/cli/helpers.go`

**Interfaces:**
- Consumes: existing Go source with `github.com/ngct/potaco` module path
- Produces: all imports pointing to `github.com/ncxton/potaco`

- [ ] **Step 1: Update go.mod module path**

Change the first line of `go.mod`:

```
module github.com/ncxton/potaco
```

- [ ] **Step 2: Update main.go import**

Change the import in `main.go` from:
```go
import "github.com/ngct/potaco/internal/cli"
```
to:
```go
import "github.com/ncxton/potaco/internal/cli"
```

- [ ] **Step 3: Update internal/cli/edit.go imports**

Change the two imports:
```go
"github.com/ngct/potaco/internal/config"
"github.com/ngct/potaco/internal/provider"
```
to:
```go
"github.com/ncxton/potaco/internal/config"
"github.com/ncxton/potaco/internal/provider"
```

- [ ] **Step 4: Update internal/cli/config_cmd.go imports**

Change the two imports:
```go
"github.com/ngct/potaco/internal/config"
"github.com/ngct/potaco/internal/provider"
```
to:
```go
"github.com/ncxton/potaco/internal/config"
"github.com/ncxton/potaco/internal/provider"
```

- [ ] **Step 5: Update internal/cli/gen.go imports**

Change the two imports:
```go
"github.com/ngct/potaco/internal/config"
"github.com/ngct/potaco/internal/provider"
```
to:
```go
"github.com/ncxton/potaco/internal/config"
"github.com/ncxton/potaco/internal/provider"
```

- [ ] **Step 6: Update internal/cli/info.go import**

Change the import:
```go
img "github.com/ngct/potaco/internal/image"
```
to:
```go
img "github.com/ncxton/potaco/internal/image"
```

- [ ] **Step 7: Update internal/cli/edit_mask.go import**

Change the import:
```go
img "github.com/ngct/potaco/internal/image"
```
to:
```go
img "github.com/ncxton/potaco/internal/image"
```

- [ ] **Step 8: Update internal/cli/config_cmd_test.go import**

Change the import:
```go
"github.com/ngct/potaco/internal/config"
```
to:
```go
"github.com/ncxton/potaco/internal/config"
```

- [ ] **Step 9: Update internal/cli/helpers.go imports**

Change the three imports:
```go
"github.com/ngct/potaco/internal/config"
img "github.com/ngct/potaco/internal/image"
"github.com/ngct/potaco/internal/provider"
```
to:
```go
"github.com/ncxton/potaco/internal/config"
img "github.com/ncxton/potaco/internal/image"
"github.com/ncxton/potaco/internal/provider"
```

- [ ] **Step 10: Run go mod tidy**

Run: `cd /home/ngct/Projects/potaco && go mod tidy`
Expected: no errors, go.sum may update or remain the same (dependencies unchanged, only module path changed)

- [ ] **Step 11: Verify build passes**

Run: `cd /home/ngct/Projects/potaco && go build ./...`
Expected: compiles successfully with no errors

- [ ] **Step 12: Verify tests pass**

Run: `cd /home/ngct/Projects/potaco && go test ./... -v`
Expected: all tests pass

- [ ] **Step 13: Verify vet passes**

Run: `cd /home/ngct/Projects/potaco && go vet ./...`
Expected: no issues

- [ ] **Step 14: Verify gofmt**

Run: `cd /home/ngct/Projects/potaco && gofmt -l .`
Expected: no output (all files formatted)

- [ ] **Step 15: Verify no remaining ngct references in source**

Run: `cd /home/ngct/Projects/potaco && grep -r "github.com/ngct/potaco" --include="*.go" --include="go.mod" .`
Expected: no output

- [ ] **Step 16: Add git remote**

Run: `cd /home/ngct/Projects/potaco && git remote add origin https://github.com/ncxton/potaco.git`
Expected: creates remote `origin`

Verify: `cd /home/ngct/Projects/potaco && git remote -v`
Expected output:
```
origin  https://github.com/ncxton/potaco.git (fetch)
origin  https://github.com/ncxton/potaco.git (push)
```

- [ ] **Step 17: Commit**

Run: `cd /home/ngct/Projects/potaco && git add go.mod go.sum main.go internal/cli/edit.go internal/cli/config_cmd.go internal/cli/gen.go internal/cli/info.go internal/cli/edit_mask.go internal/cli/config_cmd_test.go internal/cli/helpers.go && git commit -m "fix: update module path to github.com/ncxton/potaco"`
Expected: commit created

---

### Task 2: Create the CI workflow

**Files:**
- Create: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: Go 1.26 toolchain on the runner
- Produces: a workflow that validates PRs and pushes to main

- [ ] **Step 1: Create .github/workflows directory**

Run: `mkdir -p /home/ngct/Projects/potaco/.github/workflows`
Expected: directory created

- [ ] **Step 2: Write ci.yml**

Create `.github/workflows/ci.yml` with this content:

```yaml
name: CI

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - name: Download modules
        run: go mod download

      - name: Build
        run: go build ./...

      - name: Vet
        run: go vet ./...

      - name: Check gofmt
        run: |
          if [ -n "$(gofmt -l .)" ]; then
            echo "::error::files need gofmt"
            gofmt -l .
            exit 1
          fi

      - name: Test
        run: go test ./... -v
```

- [ ] **Step 3: Validate YAML syntax**

Run: `cd /home/ngct/Projects/potaco && python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))" 2>&1 || echo "python3 not available, skipping yaml validation"`
Expected: no error (or "python3 not available" which is acceptable)

- [ ] **Step 4: Commit**

Run: `cd /home/ngct/Projects/potaco && git add .github/workflows/ci.yml && git commit -m "ci: add workflow for build, vet, gofmt, and test on PRs"`
Expected: commit created

---

### Task 3: Create the GoReleaser config

**Files:**
- Create: `.goreleaser.yaml`

**Interfaces:**
- Consumes: Go 1.26 toolchain, the potaco source code
- Produces: build artifacts for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64; tar.gz archives with install.sh, LICENSE, README.md; checksums file; auto-generated changelog

- [ ] **Step 1: Write .goreleaser.yaml**

Create `/home/ngct/Projects/potaco/.goreleaser.yaml` with this content:

```yaml
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
version: 2

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
  - formats: [tar.gz]
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

- [ ] **Step 2: Validate config (if goreleaser is installed)**

Run: `cd /home/ngct/Projects/potaco && which goreleaser && goreleaser check || echo "goreleaser not installed locally, will validate in CI"`
Expected: either "Config is valid" or "goreleaser not installed locally, will validate in CI"

- [ ] **Step 3: Commit**

Run: `cd /home/ngct/Projects/potaco && git add .goreleaser.yaml && git commit -m "ci: add goreleaser v2 config for cross-platform releases"`
Expected: commit created

---

### Task 4: Create the release workflow

**Files:**
- Create: `.github/workflows/release.yml`

**Interfaces:**
- Consumes: git tags matching `v*`, the GoReleaser config from Task 3
- Produces: GitHub releases with binary archives, checksums, and auto-generated changelog

- [ ] **Step 1: Write release.yml**

Create `/home/ngct/Projects/potaco/.github/workflows/release.yml` with this content:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 2: Validate YAML syntax**

Run: `cd /home/ngct/Projects/potaco && python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))" 2>&1 || echo "python3 not available, skipping yaml validation"`
Expected: no error

- [ ] **Step 3: Commit**

Run: `cd /home/ngct/Projects/potaco && git add .github/workflows/release.yml && git commit -m "ci: add release workflow triggered by tag pushes"`
Expected: commit created

---

### Task 5: Create the MIT LICENSE

**Files:**
- Create: `LICENSE`

**Interfaces:**
- Consumes: nothing
- Produces: MIT license file with 2026 copyright, holder ncxton

- [ ] **Step 1: Write LICENSE**

Create `/home/ngct/Projects/potaco/LICENSE` with this content:

```
MIT License

Copyright (c) 2026 ncxton

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 2: Commit**

Run: `cd /home/ngct/Projects/potaco && git add LICENSE && git commit -m "docs: add MIT license"`
Expected: commit created

---

### Task 6: Create the CONTRIBUTING.md

**Files:**
- Create: `CONTRIBUTING.md`

**Interfaces:**
- Consumes: existing project conventions from AGENTS.md
- Produces: a contributing guide for open-source contributors

- [ ] **Step 1: Write CONTRIBUTING.md**

Create `/home/ngct/Projects/potaco/CONTRIBUTING.md` with this content:

````markdown
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
````

- [ ] **Step 2: Commit**

Run: `cd /home/ngct/Projects/potaco && git add CONTRIBUTING.md && git commit -m "docs: add contributing guide"`
Expected: commit created

---

### Task 7: Create the README.md

**Files:**
- Create: `README.md`

**Interfaces:**
- Consumes: project structure, subcommand flags, provider presets, config format
- Produces: a full README with banner, badges, features, install, usage, config

- [ ] **Step 1: Write README.md**

Create `/home/ngct/Projects/potaco/README.md` with this content:

````markdown
```
 ____                    ____    _(PATH)
|  _ \ _____   _____ _  |  _ \  | |      __ _ ___ _ __ ___  
| |_) / _ \ \ / / __ \ | |_) | | |____ / _` / __| '_ \ _ \ 
|  _ <  __/\ V / (__ | |  _ <  | |__|| | (_| (__| |_) | |
|_| \_\___| \_/ \___| |_| \_\ | |    \__,_\___| .__/|_|
                               |_|               |_|
                    Terminal image generation & editing CLI
```

> **Note:** The ASCII banner above may not render correctly in all markdown viewers. The tool itself is a polished CLI for image generation and editing from the terminal.

# Potaco

[![CI](https://github.com/ncxton/potaco/actions/workflows/ci.yml/badge.svg)](https://github.com/ncxton/potaco/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ncxton/potaco.svg)](https://pkg.go.dev/github.com/ncxton/potaco)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Latest Release](https://img.shields.io/github/v/release/ncxton/potaco)](https://github.com/ncxton/potaco/releases/latest)

Potaco is a Go CLI for image generation and editing via any OpenAI-compatible provider. Connect to OpenAI, Together AI, FAL, or any provider supporting `/v1/images/generations` and `/v1/images/edits`.

## Features

- **Generate** images from text prompts with size, quality, style, and seed control
- **Edit** existing images with inpainting (mask-based) and outpainting (canvas extension)
- **Config** manage provider settings with presets for OpenAI, Together AI, and FAL
- **Info** inspect image metadata (dimensions, format, file size, color model)
- **Retry** with exponential backoff on 429 and 5xx errors
- **Terminal display** via iTerm2, Kitty, and Sixel protocols
- **JSON output** for scripting and piping
- **Dry-run mode** to validate requests without calling the API
- **Provider-agnostic** works with any OpenAI-compatible image API

## Installation

### One-liner (interactive installer)

```sh
curl -fsSL https://github.com/ncxton/potaco/releases/latest/download/install.sh | sh
```

The installer detects your platform, downloads the matching binary, verifies the checksum, and installs to `/usr/local/bin` (or `~/.local/bin` as fallback). It shows a clean interactive TUI by default.

### One-liner (non-interactive, for CI/automation)

```sh
curl -fsSL https://github.com/ncxton/potaco/releases/latest/download/install.sh | POTACO_NON_INTERACTIVE=1 sh
```

### Manual download

Download the archive for your platform from the [releases page](https://github.com/ncxton/potaco/releases/latest), extract, and move the `potaco` binary to a directory in your `PATH`.

Supported platforms:
- Linux x86_64 (amd64)
- Linux ARM64 (arm64)
- macOS Intel (darwin/amd64)
- macOS Apple Silicon (darwin/arm64)

## Quick Start

```sh
# Set up your provider credentials
potaco config set --base-url https://api.openai.com --api-key sk-xxx --model dall-e-3

# Generate an image
potaco gen --prompt "a red fox in a forest"

# View it in the terminal (if supported)
potaco gen --prompt "a cityscape at night" --view
```

## Usage

### `potaco gen` -- Generate images

Generate new images from a text prompt.

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--prompt`, `-p` | string | yes | | Text description of the desired image(s) |
| `--model` | string | no | from config | Model to use (e.g., `dall-e-3`) |
| `--size` | string | no | `1024x1024` | Image dimensions (WxH) |
| `--quality` | string | no | `standard` | Image quality (`standard` or `hd`) |
| `--n` | int | no | `1` | Number of images to generate |
| `--style` | string | no | | Visual style (`vivid` or `natural`) |
| `--seed` | int | no | `0` | Reproducibility seed |
| `--output`, `-o` | string | no | auto | Output file path (auto: `potaco-YYYYMMDD-HHMMSS.png`) |
| `--output-format` | string | no | `png` | Output format (`png` or `jpeg`) |
| `--view` | bool | no | `false` | Attempt terminal image display |
| `--stdout` | bool | no | `false` | Pipe raw image bytes to stdout |
| `--dry-run` | bool | no | `false` | Print request payload without calling API |
| `--json` | bool | no | `false` | Output JSON metadata to stdout |

**Examples:**

```sh
potaco gen --prompt "a red fox in a forest"
potaco gen --prompt "a cityscape at night" --size 1792x1024 --quality hd --n 2
potaco gen --prompt "portrait of a woman" --style vivid --seed 42 --json
potaco gen --prompt "test" --dry-run
```

### `potaco edit` -- Edit existing images

Edit an existing image with inpainting (mask-based editing) or outpainting (canvas extension).

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--prompt`, `-p` | string | yes | | Text description of the edit |
| `--image` | string | yes | | Path to source image file |
| `--mask` | string | no | | Path to mask image file (white=edit, black=keep) |
| `--mask-rect` | string | no | | Rectangular mask: `x,y,w,h` in pixels |
| `--mask-circle` | string | no | | Circular mask: `x,y,r` in pixels |
| `--extend` | string | no | | Outpaint: `top=N,bottom=N,left=N,right=N` or `all=N` |
| `--output`, `-o` | string | no | auto | Output file path |
| `--dry-run` | bool | no | `false` | Print request payload without calling API |

**Examples:**

```sh
potaco edit --prompt "make it look like a painting" --image photo.png
potaco edit --prompt "remove the person" --image photo.png --mask mask.png
potaco edit --prompt "replace with a tree" --image photo.png --mask-rect 100,200,300,300
potaco edit --prompt "extend the landscape" --image photo.png --extend right=256,bottom=256
```

### `potaco config` -- Provider configuration

Manage provider settings stored in `~/.potaco/config.yaml`.

```sh
potaco config set --base-url https://api.openai.com --api-key sk-xxx --model dall-e-3
potaco config set --provider openai          # apply preset defaults
potaco config set --retries 3 --timeout 120s
potaco config show                            # display current config
potaco config list-providers                  # list available presets
```

**Provider presets:**

| Provider | Base URL | Default Model | Sizes |
|----------|----------|---------------|-------|
| `openai` | `https://api.openai.com` | `dall-e-3` | `1024x1024`, `1792x1024`, `1024x1792` |
| `together` | `https://api.together.ai` | `black-forest-labs/flux-1` | `1024x1024` |
| `fal` | `https://fal.run` | `fal-ai/flux` | `1024x1024` |

### `potaco info` -- Image metadata

Print metadata about an image file.

```sh
potaco info output.png
potaco info output.png --json
```

## Configuration

Potaco reads configuration from `~/.potaco/config.yaml`. Values are resolved with this precedence:

**CLI flags > environment variables > config file > defaults**

**Config file format** (`~/.potaco/config.yaml`):

```yaml
default:
  base_url: https://api.openai.com
  api_key: sk-xxx
  model: dall-e-3
  retries: 3
  timeout: 120s
```

**Environment variables:**

| Variable | Equivalent flag |
|----------|----------------|
| `POTACO_BASE_URL` | `--base-url` |
| `POTACO_API_KEY` | `--api-key` |
| `POTACO_MODEL` | `--model` |
| `POTACO_RETRIES` | `--retries` |
| `POTACO_TIMEOUT` | `--timeout` |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding standards, and the pull request process.

## License

[MIT](LICENSE) - Copyright (c) 2026 ncxton
````

- [ ] **Step 2: Commit**

Run: `cd /home/ngct/Projects/potaco && git add README.md && git commit -m "docs: add README with install, usage, and configuration guide"`
Expected: commit created

---

### Task 8: Create the installer script

**Files:**
- Create: `install.sh`

**Interfaces:**
- Consumes: GitHub releases at `github.com/ncxton/potaco/releases`
- Produces: an installed `potaco` binary in `/usr/local/bin` or `~/.local/bin`

- [ ] **Step 1: Write install.sh**

Create `/home/ngct/Projects/potaco/install.sh` with this content:

```sh
#!/bin/sh
set -eu

# Potaco installer - downloads and installs the potaco CLI
# Interactive by default. Set POTACO_NON_INTERACTIVE=1 for silent mode.

# ============================================================================
# Constants
# ============================================================================
REPO="ncxton/potaco"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"
GITHUB_BASE="https://github.com/${REPO}"

# Color codes (stripped if NO_COLOR or TERM=dumb)
if [ "${NO_COLOR:-}" ] || [ "${TERM:-}" = "dumb" ] || [ ! -t 1 ]; then
    CYAN=""
    GREEN=""
    YELLOW=""
    RED=""
    BOLD=""
    RESET=""
else
    CYAN="\033[36m"
    GREEN="\033[32m"
    YELLOW="\033[33m"
    RED="\033[31m"
    BOLD="\033[1m"
    RESET="\033[0m"
fi

NON_INTERACTIVE="${POTACO_NON_INTERACTIVE:-0}"

# ============================================================================
# Output helpers
# ============================================================================

info() {
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf '%s\n' "$1"
    else
        printf "${CYAN}%s${RESET}\n" "$1"
    fi
}

warn() {
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf '%s\n' "$1" >&2
    else
        printf "${YELLOW}%s${RESET}\n" "$1" >&2
    fi
}

error() {
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf '%s\n' "$1" >&2
    else
        printf "${RED}%s${RESET}\n" "$1" >&2
    fi
}

success() {
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf '%s\n' "$1"
    else
        printf "${GREEN}%s${RESET}\n" "$1"
    fi
}

# Print a horizontal line of + and - characters
print_line() {
    width="${1:-60}"
    line=""
    i=0
    while [ "$i" -lt "$width" ]; do
        line="${line}+="
        i=$((i + 1))
    done
    printf '%s\n' "$line"
}

# Print a bordered box with a title and content lines
# Usage: print_box "Title" "line1" "line2" ...
print_box() {
    title="$1"
    shift
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf '%s\n' "$title"
        for line in "$@"; do
            printf '  %s\n' "$line"
        done
        return
    fi
    # Find the longest line for box width
    max_len=${#title}
    for line in "$@"; do
        len=${#line}
        if [ "$len" -gt "$max_len" ]; then
            max_len=$len
        fi
    done
    width=$((max_len + 4))
    # Top border
    top=""
    i=0
    while [ "$i" -lt "$width" ]; do
        top="${top}+-"
        i=$((i + 2))
    done
    printf "${CYAN}%s${RESET}\n" "$top"
    # Title
    printf "${CYAN}${BOLD}+ %s${RESET}\n" "$title"
    # Content
    for line in "$@"; do
        printf "${CYAN}+${RESET} %s\n" "$line"
    done
    # Bottom border
    printf "${CYAN}%s${RESET}\n" "$top"
}

# ============================================================================
# Banner
# ============================================================================

print_banner() {
    if [ "$NON_INTERACTIVE" = "1" ]; then
        return
    fi
    cat <<'BANNER'
 ____                    ____    _   
|  _ \ _____   _____ _  |  _ \  | |      
| |_) / _ \ \ / / __ \ | |_) | | |____ 
|  _ <  __/\ V / (__ | |  _ <  | |__|| 
|_| \_\___| \_/ \___| |_| \_\ | |    
                                 |_|    
BANNER
    printf "\n"
    info "Terminal image generation & editing CLI"
    printf "\n"
}

# ============================================================================
# Spinner
# ============================================================================

SPINNER_PID=""
SPINNER_RUNNING=0
SPINNER_CHARS="|/-\\"
SPINNER_IDX=0

spinner_start() {
    if [ "$NON_INTERACTIVE" = "1" ] || [ "${TERM:-}" = "dumb" ] || [ ! -t 2 ]; then
        return
    fi
    msg="$1"
    (
        i=0
        while true; do
            char=$(printf '%s' "$SPINNER_CHARS" | cut -c$((i % 4 + 1)))
            printf "\r${CYAN}[%s]${RESET} %s   " "$char" "$msg" >&2
            sleep 0.1
            i=$((i + 1))
        done
    ) 2>/dev/null &
    SPINNER_PID=$!
    SPINNER_RUNNING=1
}

spinner_stop() {
    if [ "$SPINNER_RUNNING" = "1" ] && [ -n "$SPINNER_PID" ]; then
        kill "$SPINNER_PID" 2>/dev/null || true
        wait "$SPINNER_PID" 2>/dev/null || true
        printf "\r\033[K" >&2
        SPINNER_RUNNING=0
    fi
}

# ============================================================================
# Platform detection
# ============================================================================

detect_platform() {
    os=$(uname -s)
    arch=$(uname -m)

    case "$os" in
        Darwin) os="apple-darwin" ;;
        Linux)  os="linux" ;;
        *)
            error "Unsupported operating system: $os"
            error "Potaco supports macOS and Linux only."
            exit 1
            ;;
    esac

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)
            error "Unsupported architecture: $arch"
            error "Potaco supports amd64 (x86_64) and arm64 (aarch64) only."
            exit 1
            ;;
    esac

    printf '%s/%s' "$os" "$arch"
}

# ============================================================================
# Version detection
# ============================================================================

detect_version() {
    # Try to get the latest release tag from GitHub API
    if command -v curl >/dev/null 2>&1; then
        response=$(curl -fsSL "$GITHUB_API" 2>/dev/null || true)
    elif command -v wget >/dev/null 2>&1; then
        response=$(wget -qO- "$GITHUB_API" 2>/dev/null || true)
    else
        response=""
    fi

    if [ -n "$response" ]; then
        # Parse tag_name from JSON without jq
        tag=$(printf '%s' "$response" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')
        if [ -n "$tag" ]; then
            printf '%s' "$tag"
            return
        fi
    fi

    # Fallback: empty string means use the "latest" redirect URLs
    printf ''
}

# ============================================================================
# Download helper
# ============================================================================

# Download a URL to a file with retry
# Usage: download_file "url" "output_path"
download_file() {
    url="$1"
    output="$2"
    attempts=0
    max_attempts=3

    while [ "$attempts" -lt "$max_attempts" ]; do
        attempts=$((attempts + 1))
        if command -v curl >/dev/null 2>&1; then
            if curl -fsSL -o "$output" "$url" 2>/dev/null; then
                return 0
            fi
        elif command -v wget >/dev/null 2>&1; then
            if wget -q -O "$output" "$url" 2>/dev/null; then
                return 0
            fi
        else
            error "Neither curl nor wget is available."
            error "Install one of them to use the potaco installer."
            exit 1
        fi

        if [ "$attempts" -lt "$max_attempts" ]; then
            warn "Download failed (attempt $attempts/$max_attempts), retrying..."
            sleep 2
        fi
    done

    error "Failed to download after $max_attempts attempts."
    error "URL: $url"
    return 1
}

# ============================================================================
# Checksum verification
# ============================================================================

# Verify the downloaded tarball against the checksums file
# Usage: verify_checksum "tarball_path" "checksums_path" "tarball_filename"
verify_checksum() {
    tarball="$1"
    checksums="$2"
    tarball_name="$3"

    if [ ! -f "$checksums" ]; then
        warn "Checksums file not found, skipping verification."
        return 0
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$tarball" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$tarball" | awk '{print $1}')
    else
        warn "Neither sha256sum nor shasum is available, skipping verification."
        return 0
    fi

    # Find the matching line in the checksums file
    expected=$(grep "$tarball_name" "$checksums" | awk '{print $1}' | head -1)

    if [ -z "$expected" ]; then
        warn "Checksum for $tarball_name not found in checksums file, skipping verification."
        return 0
    fi

    if [ "$actual" != "$expected" ]; then
        error "Checksum verification failed!"
        error "Expected: $expected"
        error "Actual:   $actual"
        return 1
    fi

    return 0
}

# ============================================================================
# Main installation flow
# ============================================================================

main() {
    # Check for required tools
    if ! command -v tar >/dev/null 2>&1; then
        error "tar is required but not found in PATH."
        error "Install tar to use the potaco installer."
        exit 1
    fi

    # Print banner
    print_banner

    # Detect platform
    platform=$(detect_platform)
    os=$(printf '%s' "$platform" | cut -d/ -f1)
    arch=$(printf '%s' "$platform" | cut -d/ -f2)

    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Detected: %s\n' "$platform"
    else
        print_box "Platform" "OS: $os" "Arch: $arch"
        printf "\n"
    fi

    # Detect version
    version=$(detect_version)

    # Determine download URLs
    if [ -n "$version" ]; then
        # Version known from API
        tarball_name="potaco_${version}_${os}_${arch}.tar.gz"
        tarball_url="${GITHUB_BASE}/releases/download/${version}/${tarball_name}"
        checksums_name="potaco_${version}_checksums.txt"
        checksums_url="${GITHUB_BASE}/releases/download/${version}/${checksums_name}"
        version_display="$version"
    else
        # Fallback: use latest redirect
        tarball_name="potaco_${os}_${arch}.tar.gz"
        tarball_url="${GITHUB_BASE}/releases/latest/download/${tarball_name}"
        checksums_name="potaco_checksums.txt"
        checksums_url="${GITHUB_BASE}/releases/latest/download/${checksums_name}"
        version_display="latest"
    fi

    # Show version info
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Installing potaco %s...\n' "$version_display"
    else
        info "Installing potaco $version_display..."
        printf "\n"
    fi

    # Create temp directory
    tmpdir=$(mktemp -d 2>/dev/null || mktemp -d -t potaco-install)
    tarball_path="${tmpdir}/${tarball_name}"
    checksums_path="${tmpdir}/${checksums_name}"
    cleanup() {
        rm -rf "$tmpdir" 2>/dev/null || true
    }
    trap cleanup EXIT INT TERM

    # Download tarball
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Downloading...\n'
    else
        spinner_start "Downloading..."
    fi

    if ! download_file "$tarball_url" "$tarball_path"; then
        spinner_stop
        exit 1
    fi

    spinner_stop

    # Download checksums
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Downloading checksums...\n'
    else
        spinner_start "Downloading checksums..."
    fi

    download_file "$checksums_url" "$checksums_path" 2>/dev/null || true
    spinner_stop

    # Verify checksum
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Verifying checksum...\n'
    else
        spinner_start "Verifying checksum..."
    fi

    if ! verify_checksum "$tarball_path" "$checksums_path" "$tarball_name"; then
        spinner_stop
        error "Checksum verification failed. Aborting installation."
        exit 1
    fi
    spinner_stop

    # Extract
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Extracting...\n'
    else
        spinner_start "Extracting..."
    fi

    tar -xzf "$tarball_path" -C "$tmpdir"
    binary_path="${tmpdir}/potaco"
    spinner_stop

    if [ ! -f "$binary_path" ]; then
        error "Binary not found in archive after extraction."
        error "Expected: $binary_path"
        exit 1
    fi

    # Determine install location
    install_dir=""
    install_path=""

    if [ -w "/usr/local/bin" ]; then
        install_dir="/usr/local/bin"
        install_path="${install_dir}/potaco"
    elif [ "$NON_INTERACTIVE" = "1" ]; then
        # Non-interactive: use ~/.local/bin without asking
        install_dir="${HOME}/.local/bin"
        mkdir -p "$install_dir" 2>/dev/null || true
        install_path="${install_dir}/potaco"
    elif command -v sudo >/dev/null 2>&1; then
        # Interactive: ask about sudo
        printf "Potaco can install to /usr/local/bin (requires sudo) or %s/.local/bin.\n" "$HOME"
        printf "Install to /usr/local/bin with sudo? [Y/n] "
        read answer
        case "$answer" in
            [Yy]*|'')
                install_dir="/usr/local/bin"
                install_path="${install_dir}/potaco"
                ;;
            *)
                install_dir="${HOME}/.local/bin"
                mkdir -p "$install_dir" 2>/dev/null || true
                install_path="${install_dir}/potaco"
                ;;
        esac
    else
        # No sudo, fall back to ~/.local/bin
        install_dir="${HOME}/.local/bin"
        mkdir -p "$install_dir" 2>/dev/null || true
        install_path="${install_dir}/potaco"
    fi

    # Install the binary
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Installing to %s...\n' "$install_path"
    else
        spinner_start "Installing..."
    fi

    if [ -w "$install_dir" ]; then
        mv "$binary_path" "$install_path"
        chmod +x "$install_path"
    elif [ "$install_dir" = "/usr/local/bin" ] && command -v sudo >/dev/null 2>&1; then
        sudo mv "$binary_path" "$install_path"
        sudo chmod +x "$install_path"
    else
        # Fallback: try mv directly, might fail
        mv "$binary_path" "$install_path" 2>/dev/null || {
            spinner_stop
            error "Cannot write to $install_dir."
            error "Try running with sudo or set POTACO_NON_INTERACTIVE=1."
            exit 1
        }
        chmod +x "$install_path" 2>/dev/null || true
    fi

    spinner_stop

    # Check if install_dir is in PATH
    case ":${PATH}:" in
        *":${install_dir}:"*) ;;
        *)
            warn "Note: $install_dir is not in your PATH."
            warn "Add it with: export PATH=\"${install_dir}:\$PATH\""
            ;;
    esac

    # Print success
    printf "\n"
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Done. Potaco installed to %s\n' "$install_path"
    else
        print_box "Success" \
            "${GREEN}Potaco installed successfully!${RESET}" \
            "" \
            "Installed to: $install_path" \
            "" \
            "Next steps:" \
            "  potaco config set --base-url <url> --api-key <key>" \
            "  potaco gen --prompt \"hello\"" \
            "" \
            "Docs: https://github.com/ncxton/potaco#readme"
    fi

    printf "\n"
    exit 0
}

main "$@"
```

- [ ] **Step 2: Make it executable**

Run: `chmod +x /home/ngct/Projects/potaco/install.sh`
Expected: no error

- [ ] **Step 3: Syntax check**

Run: `sh -n /home/ngct/Projects/potaco/install.sh`
Expected: no output (syntax valid)

- [ ] **Step 4: Test non-interactive mode with --dry-run style (syntax only)**

Run: `cd /home/ngct/Projects/potaco && POTACO_NON_INTERACTIVE=1 sh -c 'os=$(uname -s); case "$os" in Darwin) echo "apple-darwin" ;; Linux) echo "linux" ;; esac'`
Expected: prints `linux` (on this Linux system)

- [ ] **Step 5: Test platform detection logic in isolation**

Run: `cd /home/ngct/Projects/potaco && sh -c 'os=$(uname -s); arch=$(uname -m); case "$os" in Darwin) os="apple-darwin" ;; Linux) os="linux" ;; esac; case "$arch" in x86_64|amd64) arch="amd64" ;; aarch64|arm64) arch="arm64" ;; esac; printf "%s/%s\n" "$os" "$arch"'`
Expected: prints `linux/amd64` (on this x86_64 Linux system)

- [ ] **Step 6: Commit**

Run: `cd /home/ngct/Projects/potaco && git add install.sh && git commit -m "ci: add native installer script with interactive TUI and non-interactive mode"`
Expected: commit created

---

### Task 9: Final verification

**Files:**
- No new files. Verification only.

**Interfaces:**
- Consumes: all files from Tasks 1-8
- Produces: confirmation that everything builds and passes

- [ ] **Step 1: Full build verification**

Run: `cd /home/ngct/Projects/potaco && go build ./...`
Expected: no errors

- [ ] **Step 2: Full test verification**

Run: `cd /home/ngct/Projects/potaco && go test ./...`
Expected: all tests pass

- [ ] **Step 3: Full vet verification**

Run: `cd /home/ngct/Projects/potaco && go vet ./...`
Expected: no issues

- [ ] **Step 4: Full gofmt verification**

Run: `cd /home/ngct/Projects/potaco && gofmt -l .`
Expected: no output

- [ ] **Step 5: Verify no old module path remains in source**

Run: `cd /home/ngct/Projects/potaco && grep -r "github.com/ngct/potaco" --include="*.go" --include="go.mod" .`
Expected: no output

- [ ] **Step 6: Verify all new files exist**

Run: `cd /home/ngct/Projects/potaco && ls -la .github/workflows/ci.yml .github/workflows/release.yml .goreleaser.yaml install.sh README.md LICENSE CONTRIBUTING.md`
Expected: all files listed with no "No such file" errors

- [ ] **Step 7: Verify installer syntax**

Run: `sh -n /home/ngct/Projects/potaco/install.sh`
Expected: no output

- [ ] **Step 8: Verify git remote is set**

Run: `cd /home/ngct/Projects/potaco && git remote -v`
Expected: shows `origin https://github.com/ncxton/potaco.git`

- [ ] **Step 9: Verify git status is clean**

Run: `cd /home/ngct/Projects/potaco && git status --porcelain`
Expected: no output (working tree clean)

- [ ] **Step 10: Final commit (if anything remains)**

Run: `cd /home/ngct/Projects/potaco && git status --porcelain | head -1`
If output is non-empty: `cd /home/ngct/Projects/potaco && git add -A && git commit -m "fix: resolve final verification issues"`
If empty: `echo "Working tree clean, nothing to commit"`
Expected: working tree clean
