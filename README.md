<p align="center">
  <img src="assets/potaco-banner.png" alt="Potaco" width="100%">
</p>

# Potaco

[![CI](https://github.com/ncxton/potaco/actions/workflows/ci.yml/badge.svg)](https://github.com/ncxton/potaco/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ncxton/potaco.svg)](https://pkg.go.dev/github.com/ncxton/potaco)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Latest Release](https://img.shields.io/github/v/release/ncxton/potaco)](https://github.com/ncxton/potaco/releases/latest)

Potaco is a Go CLI for image generation and editing via multiple AI image providers. Connect to OpenAI, fal, Vercel AI Gateway, or any OpenAI-compatible API via the custom provider.

> [!WARNING]
> This project is still in an early stage of development. It has not been thoroughly tested yet, and critical breakages or bugs are to be expected. Use at your own risk, and please report any issues you encounter.

## Features

- **Multi-provider** support for OpenAI, fal, Vercel AI Gateway, and custom OpenAI-compatible endpoints
- **Encrypted credentials** stored locally
- **Auth management** with `auth add/remove/list` commands
- **Provider switching** with `potaco use <provider>`
- **Model discovery** via provider APIs
- **Generate** images from text prompts with size, quality, seed, and guidance control
- **Edit** existing images with inpainting (mask-based) and outpainting (canvas extension)
- **Status** and **models** commands to inspect current state and available models
- **Interactive TUI** with interactive forms for auth, model selection, and provider switching
- **Info** inspect image metadata (dimensions, format, file size, color model)
- **Retry** with exponential backoff on rate-limit and server errors
- **JSON output** for scripting and piping
- **Dry-run mode** to validate requests without calling the API
- **Non-interactive mode** with `--non-interactive` flag for CI and agents

## Installation

### One-liner (interactive installer)

```sh
curl -fsSL https://github.com/ncxton/potaco/releases/latest/download/install.sh | sh
```

The installer detects your platform, downloads the matching binary, verifies the checksum, and installs to `~/.local/bin`.

### One-liner (non-interactive)

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

# Switch to another provider
potaco auth add fal --api-key <fal-key>
potaco use fal

# List or pick available models from the active provider
potaco models                      # interactive picker (persists the selected model)
potaco models list                 # static list of models
potaco models list openai          # static list for a specific provider

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
| `--guidance-scale` | float | no | `0` | Guidance scale |
| `--negative-prompt` | string | no | | Negative prompt |
| `--response-format` | string | no | `b64_json` | Response format (`url` or `b64_json`) |
| `--output`, `-o` | string | no | auto | Output file path (auto: `potaco-YYYYMMDD-HHMMSS.png`) |
| `--output-format` | string | no | `png` | Output format (`png` or `jpeg`) |
| `--stdout` | bool | no | `false` | Pipe raw image bytes to stdout |
| `--dry-run` | bool | no | `false` | Print request payload without calling API |
| `--json` | bool | no | `false` | Output JSON metadata to stdout |
| `--provider` | string | no | from config | Override active provider |
| `--base-url` | string | no | from provider | Override API base URL |
| `--api-key` | string | no | from credentials | Override API key |
| `--retries` | int | no | `2` | Max retry attempts |
| `--timeout` | string | no | `120` | Request timeout in seconds |

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
| `--model` | string | no | from config | Model to use |
| `--size` | string | no | `1024x1024` | Image dimensions (WxH) |
| `--n` | int | no | `1` | Number of images to generate |
| `--response-format` | string | no | `b64_json` | Response format (`url` or `b64_json`) |
| `--output`, `-o` | string | no | auto | Output file path |
| `--output-format` | string | no | `png` | Output format (`png` or `jpeg`) |
| `--stdout` | bool | no | `false` | Pipe raw image bytes to stdout |
| `--dry-run` | bool | no | `false` | Print request payload without calling API |
| `--provider` | string | no | from config | Override active provider |
| `--base-url` | string | no | from provider | Override API base URL |
| `--api-key` | string | no | from credentials | Override API key |
| `--retries` | int | no | `2` | Max retry attempts |
| `--timeout` | string | no | `120` | Request timeout in seconds |

**Examples:**

```sh
potaco edit --prompt "make it look like a painting" --image photo.png
potaco edit --prompt "remove the person" --image photo.png --mask mask.png
potaco edit --prompt "replace with a tree" --image photo.png --mask-rect 100,200,300,300
potaco edit --prompt "extend the landscape" --image photo.png --extend right=256,bottom=256
```

### `potaco auth` -- Manage provider credentials

Connect providers, add API keys, and manage credentials. Credentials are encrypted at rest.

```sh
potaco auth add openai                        # interactive TUI flow
potaco auth add openai --api-key sk-xxx       # non-interactive
potaco auth add fal --api-key <key>           # connect fal
potaco auth add vercel --api-key <key>        # connect Vercel AI Gateway
potaco auth add custom --api-key <key> --base-url <url>  # connect any OpenAI-compatible endpoint
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

### `potaco models` -- Pick or list available models

Discover models from the active provider or a specified provider. By default, the command launches an interactive picker that persists the selected model to config. In non-interactive mode it falls back to a static list.

```sh
potaco models                                    # interactive picker for the active provider
potaco models openai                             # interactive picker for a specific provider
potaco models --non-interactive                  # static list for the active provider
potaco models list                               # static list for the active provider
potaco models list openai                        # static list for a specific provider
potaco models --json --non-interactive           # JSON output (static list)
```

Use `potaco models list` when you only want to see the available models without changing the active model.

### `potaco config` -- Provider settings

Manage per-provider settings stored in `~/.potaco/config.yaml`. API keys are stored separately in the encrypted credential file.

```sh
potaco config set --retries 3 --timeout 120   # set retries and timeout (seconds) for active provider
potaco config set --model gpt-image-2           # change default model
potaco config set --base-url https://api.example.com/v1  # change custom base URL for active provider
potaco config show                               # display current config
```

### `potaco info` -- Image metadata

Print metadata about an image file.

```sh
potaco info output.png
potaco info output.png --json
```

### `potaco version` -- Print version

Print the current binary version and check for updates.

```sh
potaco version
potaco version --json
potaco --version
```

### `potaco update` -- Update potaco

Download and run the installer for the latest release.

```sh
potaco update                    # update if newer version available
potaco update --force            # force update even if already latest
```

### `potaco uninstall` -- Remove potaco

Remove the potaco binary and optionally its configuration.

```sh
potaco uninstall                  # interactive removal
potaco uninstall --remove-config # also remove ~/.potaco/
potaco uninstall --yes            # skip confirmation prompts
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
    base_url: https://api.openai.com/v1
    retries: 3
    timeout: 120
  fal:
    model: fal-ai/flux/dev
    base_url: https://fal.run
    retries: 3
    timeout: 120
```

The `base_url` field is optional and overrides the preset URL for a provider. It is required for the `custom` provider because there is no preset base URL.

API keys are stored separately in `~/.potaco/credentials.enc`, encrypted with a machine-derived key.

**Environment variables:**

| Variable | Description |
|----------|-------------|
| `POTACO_PROVIDER` | Active provider name (e.g., `openai`, `fal`, `vercel`, `custom`) |
| `POTACO_API_KEY` | API key for the active provider |
| `POTACO_MODEL` | Default model for the active provider |
| `POTACO_BASE_URL` | Override the provider's base URL (required for `custom`) |
| `POTACO_RETRIES` | Max retry attempts |
| `POTACO_TIMEOUT` | Request timeout in seconds (e.g., `120`) |
| `POTACO_NON_INTERACTIVE` | Set to `1` to disable TUI flows |

**Supported providers:**

The known providers below ship with preset base URLs. The `custom` provider is also supported for any OpenAI-compatible endpoint, but it has no preset and requires `--base-url` or `base_url` in config.

| Provider | Auth Type | Edit Support |
|----------|-----------|--------------|
| `openai` | Bearer | Yes |
| `fal` | Key | Yes |
| `vercel` | Bearer | No |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding standards, and the pull request process.

### CI Checks

Every PR runs: build, go vet, gofmt, staticcheck, gocyclo (complexity threshold 30), go mod tidy, coverage, gitleaks (secret scanning), and tests.

### Pre-commit Hooks

```sh
sh scripts/install-hooks.sh   # Install gofmt, vet, tidy, and test hooks
```

### Environment Variables

See `.env.example` for a template of all supported environment variables.

## License

[MIT](LICENSE) - Copyright (c) 2026 ncxton
