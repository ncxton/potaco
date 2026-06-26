---
name: potaco
version: 1.1.0
description: |
  Use when working with the potaco CLI for terminal image generation,
  image editing, text-to-image, inpainting, outpainting, provider auth
  setup (OpenAI, fal, Vercel AI Gateway), model discovery, or debugging
  potaco command failures.
---

# Potaco CLI Usage

Potaco is a Go CLI for image generation and editing via multi-provider
adapters (OpenAI, fal, Vercel AI Gateway) with encrypted credential storage
and interactive TUI flows.

## Prerequisites: Verify potaco is installed

Before doing anything, check that potaco exists:

```sh
potaco version
```

If the command is not found, ask the user if they want to install it.
The install script lives in the repo at `install.sh` and can also be
fetched from GitHub releases:

```sh
curl -fsSL https://github.com/ncxton/potaco/releases/latest/download/install.sh | sh
```

For non-interactive installation (CI, automated environments):

```sh
curl -fsSL https://github.com/ncxton/potaco/releases/latest/download/install.sh | POTACO_NON_INTERACTIVE=1 sh
```

The binary installs to `~/.local/bin/potaco`. If `~/.local/bin` is not in
PATH, the installer offers to add it to the shell config automatically.

You can also build from source if the repo is cloned:

```sh
go build -o potaco .
```

Do not proceed with any other command until potaco is installed and working.

## Auth Setup

Before generating or editing images, a provider must be connected with an
API key. Credentials are stored encrypted at `~/.potaco/credentials.enc`.

### Connect a provider

```sh
potaco auth add openai --api-key sk-...
```

- `--api-key` can be omitted in interactive mode; a TUI prompt will appear.
- `--force` skips provider verification (useful for custom base URLs).
- `--model` overrides the default model for the provider.
- Without a provider argument in interactive mode, a provider picker launches.

Available providers: `openai`, `fal`, `vercel`.

### List connected providers

```sh
potaco auth list
potaco auth list --json
```

### Remove a provider

```sh
potaco auth remove openai
```

### Switch active provider

```sh
potaco use openai
potaco use openai --model gpt-image-2
```

Without a provider argument in interactive mode, a provider/model picker launches.

### Check status

```sh
potaco status
potaco status --json
```

Shows the active provider, active model, connected providers, and file paths.

Always make sure at least one provider is connected before running gen or edit.
If no provider is configured, the command will exit with a config error (code 2)
and a hint to run `potaco auth add`.

## Generation (text-to-image)

```sh
potaco gen --prompt "a cat in a spacesuit"
```

### Key flags

| Flag | Default | Description |
|------|---------|-------------|
| `--prompt` / `-p` | (required) | Text description of the desired image |
| `--model` | active model | Model to use |
| `--size` | `1024x1024` | Image dimensions (WxH) |
| `--quality` | `auto` | Quality: low, medium, high, or auto |
| `--n` | `1` | Number of images to generate |
| `--seed` | `0` | Reproducibility seed |
| `--output` / `-o` | auto-generated | Output file path |
| `--output-format` | `png` | Output format: png or jpeg |
| `--stdout` | false | Pipe raw image bytes to stdout |
| `--dry-run` | false | Print request payload without calling the API |

### Provider override flags

| Flag | Env var | Description |
|------|---------|-------------|
| `--provider` | `POTACO_PROVIDER` | Provider preset: openai, fal, vercel |
| `--api-key` | `POTACO_API_KEY` | Override API key |
| `--base-url` | `POTACO_BASE_URL` | Override API base URL |
| `--model` | `POTACO_MODEL` | Override model |
| `--retries` | `POTACO_RETRIES` | Max retry attempts |
| `--timeout` | `POTACO_TIMEOUT` | Request timeout in seconds |

Precedence: CLI flag > env var > config file > provider preset default.

### Output modes

- **Default**: saves to an auto-generated filename, prints `Saved to: <path>`.
- **`-o path.png`**: saves to the specified path. If `--n > 1`, appends `-0`, `-1`, etc.
- **`--stdout`**: pipes raw image bytes to stdout. Only works with `--n 1`.
  Use `--output-format jpeg` for JPEG output.
- **`--json`**: prints JSON metadata (path, dimensions, model, params, latency) to stdout.

### Dry run

```sh
potaco gen --prompt "test" --dry-run
```

Prints the method, URL, headers (api key redacted), and request body as JSON.
Useful for debugging request payloads without spending API credits.

## Image Editing

```sh
potaco edit --prompt "add a hat" --image photo.png
```

Edit takes a source image and a text prompt. Three edit modes are available
based on mask flags. For details on inpainting and outpainting, load the
`references/editing.md` reference.

### Basic edit (no mask)

```sh
potaco edit --prompt "make it night time" --image input.png -o output.png
```

The provider applies the prompt to the entire image.

### Common edit flags

| Flag | Description |
|------|-------------|
| `--prompt` / `-p` | (required) Text description of the edit |
| `--image` | (required) Path to source image |
| `--mask` | Path to mask image file (white=edit, black=keep) |
| `--mask-rect` | Rectangular mask: `x,y,w,h` in pixels |
| `--mask-circle` | Circular mask: `x,y,r` in pixels |
| `--extend` | Outpaint: `top=N,bottom=N,left=N,right=N` or `all=N` |
| `--output` / `-o` | Output file path |
| `--stdout` | Pipe raw image bytes to stdout |
| `--dry-run` | Print request payload without calling the API |

Note: the Vercel AI Gateway provider does not support image editing. If the
active provider is vercel, the edit command returns a clear error.

## Model Discovery

```sh
potaco models                    # list models for active provider
potaco models openai             # list models for a specific provider
potaco models --params gpt-image-2  # show supported parameters for a model
potaco models --json
```

## Utility Commands

### Get image metadata

```sh
potaco info path/to/image.png
potaco info image.png --json
```

### Version and updates

```sh
potaco version
potaco version --json
potaco update
potaco update --force
```

### Uninstall

```sh
potaco uninstall
potaco uninstall --remove-config
potaco uninstall -y                # skip confirmation in interactive mode
potaco uninstall -y --remove-config
```

## Persistent Flags

These flags work on all commands:

| Flag | Description |
|------|-------------|
| `--json` | Output JSON metadata to stdout |
| `--verbose` | Print retry attempts and debug info to stderr |
| `--non-interactive` | Force non-interactive mode (skip all TUI) |

The `POTACO_NON_INTERACTIVE=1` env var achieves the same as `--non-interactive`.

## Quick Reference: Common Workflows

### First-time setup

```sh
potaco auth add openai --api-key sk-...
potaco gen --prompt "hello world"
```

### Switch providers and generate

```sh
potaco use fal
potaco gen --prompt "a sunset over the ocean"
```

### Generate to stdout and pipe to a file

```sh
potaco gen --prompt "a logo" --stdout --output-format png > logo.png
```

### Batch generate multiple images

```sh
potaco gen --prompt "various cats" --n 4 -o cats.png
# produces cats-0.png, cats-1.png, cats-2.png, cats-3.png
```

### Debug a request without spending credits

```sh
potaco gen --prompt "test image" --dry-run
potaco edit --prompt "test edit" --image input.png --dry-run
```

## Common Mistakes

- **Running gen/edit without a provider**: Returns exit code 2. Always run
  `potaco auth add <provider> --api-key <key>` first.
- **Using `--stdout` with `--n > 1`**: Returns an image error. Stdout mode
  requires a single image. Use `--output` for multiple images.
- **Editing with the Vercel provider**: Vercel AI Gateway does not support
  image editing. Switch with `potaco use openai` or `potaco use fal`.
- **Forgetting `--non-interactive` in scripts/CI**: Without it, commands
  that need a TTY will hang. Set `POTACO_NON_INTERACTIVE=1` or pass
  `--non-interactive`.
- **Wrong model name**: Causes API errors. Run `potaco models` to list
  available models, or `potaco models --params <model>` to check parameters.
- **Output path is a directory**: Returns an image error. `--output` expects
  a filename, not a directory. Omit `-o` to auto-generate a filename.
- **Multiple mask flags at once**: `--mask`, `--mask-rect`, and `--mask-circle`
  are not combined. `--mask` takes priority, then `--mask-rect`, then
  `--mask-circle`. Use only one per command.

## Advanced References

Load these files for detailed guidance on advanced topics:

- **`references/providers.md`** - Provider-specific details: default models,
  base URLs, auth headers, fallback model lists, edit support matrix.
- **`references/editing.md`** - Editing modes in depth: basic edit, inpainting
  with masks (file, rect, circle), outpainting with extend, temp file cleanup.
- **`references/configuration.md`** - Config file format, env vars, full
  precedence chain, retries/timeouts, `config set/show` commands.
- **`references/troubleshooting.md`** - Exit codes, error message patterns,
  debug log location, common failure scenarios and fixes.
