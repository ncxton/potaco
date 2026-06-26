<p align="center">
  <img src="assets/potaco-banner.png" alt="Potaco" width="100%">
</p>

# Potaco

[![CI](https://github.com/ncxton/potaco/actions/workflows/ci.yml/badge.svg)](https://github.com/ncxton/potaco/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ncxton/potaco.svg)](https://pkg.go.dev/github.com/ncxton/potaco)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Latest Release](https://img.shields.io/github/v/release/ncxton/potaco)](https://github.com/ncxton/potaco/releases/latest)

Potaco is a Go CLI for image generation and editing via multiple AI image providers. Connect to OpenAI, fal, or the Vercel AI Gateway with per-provider adapters that handle different API shapes, auth methods, and model discovery.

> [!WARNING]
> This project is still in an early stage of development. It has not been thoroughly tested yet, and critical breakages or bugs are to be expected. Use at your own risk, and please report any issues you encounter.

## Features

- **Multi-provider** support for OpenAI, fal, and Vercel AI Gateway via adapter interface
- **Encrypted credentials** stored locally with AES-256-GCM encryption (machine-derived key)
- **Auth management** with `auth add/remove/list` commands
- **Provider switching** with `potaco use <provider>`
- **Model discovery** via provider APIs with hardcoded fallbacks
- **Generate** images from text prompts with size, quality, style, and seed control
- **Edit** existing images with inpainting (mask-based) and outpainting (canvas extension)
- **Status** and **models** commands to inspect current state and available models
- **Interactive TUI** with Bubbletea/huh forms for auth, model selection, and provider switching
- **Info** inspect image metadata (dimensions, format, file size, color model)
- **Retry** with exponential backoff on 429 and 5xx errors
- **Terminal display** via iTerm2, Kitty, and Sixel protocols
- **JSON output** for scripting and piping
- **Dry-run mode** to validate requests without calling the API
- **Non-interactive mode** with `--non-interactive` flag for CI and agents

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
# Connect a provider (interactive TUI prompts for key, verification, and model)
potaco auth add openai

# Or non-interactive:
potaco auth add openai --api-key sk-xxx

# Generate an image
potaco gen --prompt "a red fox in a forest"

# View it in the terminal (if supported)
potaco gen --prompt "a cityscape at night" --view

# Switch to another provider
potaco auth add fal --api-key <fal-key>
potaco use fal

# List available models from the active provider
potaco models

# Check current status
potaco status
```

## Usage

### `potaco gen` -- Generate images

Generate new images from a text prompt.

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--prompt`, `-p` | string | yes | | Text description of the desired image(s) |
| `--model` | string | no | from config | Model to use (e.g., `gpt-image-2`) |
| `--size` | string | no | `1024x1024` | Image dimensions (WxH) |
| `--quality` | string | no | `auto` | Image quality (`low`, `medium`, `high`, or `auto`) |
| `--n` | int | no | `1` | Number of images to generate |
| `--seed` | int | no | `0` | Reproducibility seed |
| `--output`, `-o` | string | no | auto | Output file path (auto: `potaco-YYYYMMDD-HHMMSS.png`) |
| `--output-format` | string | no | `png` | Output format (`png` or `jpeg`) |
| `--view` | bool | no | `false` | Attempt terminal image display |
| `--stdout` | bool | no | `false` | Pipe raw image bytes to stdout |
| `--dry-run` | bool | no | `false` | Print request payload without calling API |
| `--json` | bool | no | `false` | Output JSON metadata to stdout |
| `--provider` | string | no | from config | Override active provider |
| `--base-url` | string | no | from adapter | Override API base URL |
| `--api-key` | string | no | from credentials | Override API key |

**Examples:**

```sh
potaco gen --prompt "a red fox in a forest"
potaco gen --prompt "a cityscape at night" --size 1536x1024 --quality high --n 2
potaco gen --prompt "portrait of a woman" --seed 42 --json
potaco gen --prompt "test" --dry-run
potaco gen --prompt "test" --provider fal --api-key <key> --dry-run
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
| `--provider` | string | no | from config | Override active provider |
| `--base-url` | string | no | from adapter | Override API base URL |
| `--api-key` | string | no | from credentials | Override API key |

**Examples:**

```sh
potaco edit --prompt "make it look like a painting" --image photo.png
potaco edit --prompt "remove the person" --image photo.png --mask mask.png
potaco edit --prompt "replace with a tree" --image photo.png --mask-rect 100,200,300,300
potaco edit --prompt "extend the landscape" --image photo.png --extend right=256,bottom=256
```

### `potaco auth` -- Manage provider credentials

Connect providers, add API keys, and manage credentials. Credentials are encrypted at rest with AES-256-GCM.

```sh
potaco auth add openai                        # interactive TUI flow
potaco auth add openai --api-key sk-xxx       # non-interactive
potaco auth add fal --api-key <key>           # connect fal
potaco auth add vercel --api-key <key>        # connect Vercel AI Gateway
potaco auth remove openai                     # disconnect a provider
potaco auth list                               # list connected providers
potaco auth list --json                        # JSON output
```

### `potaco use` -- Switch active provider

Switch the active provider and optionally change the model.

```sh
potaco use openai                              # switch to openai
potaco use fal                                 # switch to fal
potaco use openai --model gpt-image-2          # switch and set model
```

When run interactively with no arguments, a TUI picker appears.

### `potaco status` -- Show current status

Display the active provider, model, config and credential paths, and connected providers.

```sh
potaco status                                  # text output
potaco status --json                            # JSON output
```

### `potaco models` -- List available models

Query the active (or specified) provider for available models and their parameters.

```sh
potaco models                                   # list models from active provider
potaco models --params gpt-image-2               # show parameters for a model
potaco models --json                             # JSON output
potaco models openai                             # list models from specific provider
```

### `potaco config` -- Provider settings

Manage per-provider settings stored in `~/.potaco/config.yaml`. API keys are stored separately in the encrypted credential file.

```sh
potaco config set --retries 3 --timeout 120s   # set retries and timeout for active provider
potaco config set --model gpt-image-2           # change default model
potaco config show                               # display current config
```

### `potaco info` -- Image metadata

Print metadata about an image file.

```sh
potaco info output.png
potaco info output.png --json
```

### `--non-interactive` flag

Disable interactive TUI flows. Useful for CI, scripts, and AI agents. Can also be set via the `POTACO_NON_INTERACTIVE=1` environment variable.

```sh
potaco --non-interactive auth add openai --api-key sk-xxx
POTACO_NON_INTERACTIVE=1 potaco models
```

## Configuration

Potaco stores configuration in `~/.potaco/config.yaml` and encrypted credentials in `~/.potaco/credentials.enc`.

**Config file format** (`~/.potaco/config.yaml`):

```yaml
active_provider: openai
active_model: gpt-image-2
providers:
  openai:
    model: gpt-image-2
    retries: 3
    timeout: 120s
  fal:
    model: fal-ai/flux/dev
    retries: 3
    timeout: 120s
```

API keys are stored separately in `~/.potaco/credentials.enc`, encrypted with AES-256-GCM using a machine-derived key.

**Environment variables:**

| Variable | Description |
|----------|-------------|
| `POTACO_PROVIDER` | Active provider name (e.g., `openai`, `fal`, `vercel`) |
| `POTACO_API_KEY` | API key for the active provider |
| `POTACO_MODEL` | Default model for the active provider |
| `POTACO_BASE_URL` | Override the provider's base URL |
| `POTACO_RETRIES` | Max retry attempts |
| `POTACO_TIMEOUT` | Request timeout (e.g., `120s`) |
| `POTACO_NON_INTERACTIVE` | Set to `1` to disable TUI flows |

**Supported providers:**

| Provider | Default Model | Auth Type | Edit Support |
|----------|---------------|-----------|--------------|
| `openai` | `gpt-image-2` | Bearer | Yes |
| `fal` | `fal-ai/flux/dev` | Key | Yes (base64) |
| `vercel` | `openai/gpt-image-2` | Bearer | No |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding standards, and the pull request process.

## License

[MIT](LICENSE) - Copyright (c) 2026 ncxton
