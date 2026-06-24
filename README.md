```
 ____                    ____      _    
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
