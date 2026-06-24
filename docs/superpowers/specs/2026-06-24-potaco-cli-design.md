# Potaco CLI Design Spec

## Overview

Potaco is a terminal CLI for advanced image generation and editing. It connects to any OpenAI-compatible provider that supports `/v1/images/generations` and `/v1/images/edits` endpoints. Built for both humans and agents, but optimized for agent usage with predictable flags, structured output, and dry-run validation.

**v0 MVP scope**: basic generation, basic edit, parameter adjustment flags, inpainting, and outpainting.

## Technology

- **Language**: Go (single static binary)
- **CLI framework**: Cobra (subcommands, auto-generated help/completions)
- **Image processing**: Go standard library `image` package, `image/png`, `image/jpeg`, `golang.org/x/image/draw` for high-quality resizing
- **Config format**: YAML (`~/.potaco/config.yaml`)

## Architecture

Layered monolith with clean one-directional package dependencies:

```
cli (top)     -> provider, config, image
provider      -> config (for types only)
config        -> (no internal deps)
image         -> (no internal deps, pure utility)
```

Each internal package has one clear purpose and can be tested independently.

### Package Layout

```
potaco/
  go.mod
  main.go
  internal/
    cli/
      root.go               # root command, global flags (--json, --verbose)
      gen.go                # gen subcommand
      edit.go               # edit subcommand
      config.go             # config subcommand
      info.go               # info subcommand
      output.go             # output formatting: text, JSON, stdout, view
    provider/
      client.go             # HTTP client: Generate, Edit methods
      types.go              # request/response types matching OpenAI API
      presets.go            # provider presets: openai, together, fal
      retry.go              # retry with backoff logic
    image/
      mask.go               # mask generation from flags or file
      canvas.go             # canvas operations: resize, paste, outpaint prep
      io.go                 # image read/write, format detection
      view.go               # terminal image display protocol detection
    config/
      config.go             # config file loading, env var override, merge logic
      types.go              # config struct definitions
```

## Command Surface

### `potaco gen` — Generate new images

Generate images from a text prompt via `POST /v1/images/generations`.

**Flags**:

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--prompt` | string | yes | — | Text description of desired image |
| `--model` | string | no | from config | Model to use (e.g., dall-e-3, dall-e-2) |
| `--size` | string | no | 1024x1024 | Image dimensions (WxH) |
| `--quality` | string | no | standard | Image quality (standard or hd) |
| `--n` | int | no | 1 | Number of images to generate |
| `--style` | string | no | "" | Visual style (vivid or natural, when supported) |
| `--seed` | int | no | 0 (unset) | Reproducibility seed (when supported) |
| `--guidance-scale` | float | no | 0 (unset) | Guidance scale (when supported) |
| `--negative-prompt` | string | no | "" | Negative prompt (when supported, passthrough) |
| `--response-format` | string | no | b64_json | Response format (url or b64_json) |
| `--output` | string | no | auto | Output file path (auto: potaco-\<timestamp\>.png) |
| `--output-format` | string | no | png | Output format (png or jpeg) |
| `--view` | bool | no | false | Attempt terminal image display |
| `--stdout` | bool | no | false | Pipe raw image bytes to stdout |
| `--provider` | string | no | from config | Provider preset (openai, together, fal) |
| `--base-url` | string | no | from config | Override API base URL |
| `--api-key` | string | no | from config | Override API key |
| `--retries` | int | no | from config | Max retry attempts on 429/5xx |
| `--timeout` | duration | no | from config | Request timeout |
| `--dry-run` | bool | no | false | Validate and print request payload without calling API |
| `--json` | bool | no | false | Output JSON metadata to stdout |
| `--verbose` | bool | no | false | Print retry attempts and debug info to stderr |

**Examples**:
```bash
potaco gen --prompt "a red fox in a forest"
potaco gen --prompt "a cityscape at night" --size 1792x1024 --quality hd --n 2
potaco gen --prompt "portrait of a woman" --style vivid --seed 42 --json
potaco gen --prompt "test" --dry-run
```

### `potaco edit` — Edit existing images

Edit an existing image via `POST /v1/images/edits`. Supports three modes auto-detected from flags:

1. **Basic edit**: `--image` + `--prompt` (restyle or re-edit the whole image)
2. **Inpaint**: `--image` + `--prompt` + mask (`--mask` file, or `--mask-rect`, or `--mask-circle`)
3. **Outpaint**: `--image` + `--prompt` + `--extend` (CLI handles canvas resize + mask internally)

**Mode detection logic**:
- If `--extend` is set: outpaint mode
- If `--mask` or `--mask-rect` or `--mask-circle` is set: inpaint mode
- Otherwise: basic edit mode

**Flags**:

All `--prompt`, `--model`, `--size`, `--quality`, `--n`, `--response-format`, `--output`, `--output-format`, `--view`, `--stdout`, `--provider`, `--base-url`, `--api-key`, `--retries`, `--timeout`, `--dry-run`, `--json`, `--verbose` flags from `gen` are available.

Additional edit-specific flags:

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--image` | string | yes | — | Path to source image file |
| `--mask` | string | no | — | Path to mask image file (white=edit, black=keep) |
| `--mask-rect` | string | no | — | Rectangular mask: x,y,w,h in pixels |
| `--mask-circle` | string | no | — | Circular mask: x,y,r in pixels (r=radius) |
| `--extend` | string | no | — | Outpaint extension: top,bottom,left,right in pixels |

**Extend flag format**:
- `--extend top=256` (only top)
- `--extend top=256,bottom=128` (multiple directions)
- `--extend all=200` (all sides equal)
- `--extend left=100,right=100` (horizontal only)

**Examples**:
```bash
# Basic edit
potaco edit --prompt "make it look like a painting" --image photo.png

# Inpaint with mask file
potaco edit --prompt "remove the person" --image photo.png --mask mask.png

# Inpaint with geometric mask
potaco edit --prompt "replace with a tree" --image photo.png --mask-rect 100,200,300,300

# Outpaint
potaco edit --prompt "extend the landscape" --image photo.png --extend right=256,bottom=256
potaco edit --prompt "more sky" --image photo.png --extend top=256
```

### `potaco config` — Provider configuration

Manage provider settings stored in `~/.potaco/config.yaml`.

**Subcommands**:

```bash
# Set individual values
potaco config set --base-url https://api.openai.com --api-key sk-xxx --model dall-e-3
potaco config set --provider openai     # applies preset defaults
potaco config set --retries 3 --timeout 120s

# Show current configuration
potaco config show

# List available provider presets
potaco config list-providers
```

### `potaco info` — Image metadata

Print metadata about any image file.

```bash
potaco info output.png
potaco info output.png --json
```

Prints: file path, format (PNG/JPEG), dimensions (WxH), file size in bytes, color model. With `--json`, structured JSON output.

## Configuration

### Config file: `~/.potaco/config.yaml`

```yaml
default:
  base_url: "https://api.openai.com"
  api_key: "sk-..."
  model: "dall-e-3"
  retries: 2
  timeout: "120s"
provider_presets:
  openai:
    base_url: "https://api.openai.com"
    default_model: "dall-e-3"
    sizes: ["1024x1024", "1792x1024", "1024x1792"]
  together:
    base_url: "https://api.together.ai"
    default_model: "black-forest-labs/flux-1"
    sizes: ["1024x1024"]
  fal:
    base_url: "https://fal.run"
    default_model: "fal-ai/flux"
    sizes: ["1024x1024"]
```

### Configuration Precedence (highest to lowest)

1. **CLI flags**: `--model`, `--base-url`, `--api-key`, `--retries`, `--timeout`
2. **Environment variables**: `POTACO_BASE_URL`, `POTACO_API_KEY`, `POTACO_MODEL`, `POTACO_RETRIES`, `POTACO_TIMEOUT`
3. **Config file**: `default` section values
4. **Provider preset defaults**: applied when `--provider` is specified and no explicit value is set at a higher precedence level

If no config file exists, potaco runs with all values from CLI flags or env vars. Missing required values (base_url, api_key) produce a clear error.

### Provider Presets

Presets bundle sensible defaults per known provider:
- `openai`: base_url, default_model, supported sizes
- `together`: base_url, default_model, supported sizes
- `fal`: base_url, default_model, supported sizes

Presets are hardcoded in `internal/provider/presets.go`. Using `--provider openai` applies the preset's base_url and default_model unless overridden by a higher-precedence source.

## Provider Client

### `internal/provider/client.go`

`Client` struct:
- Fields: `baseURL`, `apiKey`, `httpClient`, `retries`, `timeout`
- Constructor: `NewClient(cfg ClientConfig) *Client`

**`Generate(ctx, GenerateRequest) (*GenerateResponse, error)`**:
- Calls `POST {baseURL}/v1/images/generations`
- Content-Type: `application/json`
- Request body: JSON with prompt, model, size, quality, n, style, response_format, seed (when supported), guidance_scale (when supported), negative_prompt (when supported)
- Response: parse JSON, extract image data (b64_json or fetch URL)

**`Edit(ctx, EditRequest) (*EditResponse, error)`**:
- Calls `POST {baseURL}/v1/images/edits`
- Content-Type: `multipart/form-data`
- Parts: `image` (file), `mask` (file, optional), `prompt` (text), `model` (text), `size` (text), `n` (text), `response_format` (text)
- Response: same as generate

### Request/Response Types (`provider/types.go`)

Match the OpenAI API spec:
- `GenerateRequest`, `EditRequest` structs
- `ImageResponse` with `[]Image` (each having `b64_json` or `url`)
- Tags for JSON field mapping (e.g., `response_format`, `guidance_scale`)

### Retry Logic (`provider/retry.go`)

- Retries on HTTP 429 and 5xx status codes
- Exponential backoff: 1s, 2s, 4s (with jitter in range of 0-500ms)
- Max retries configurable (default 2)
- Logs retry attempts to stderr when `--verbose` is set
- Does not retry on 4xx (except 429), network errors are retried once

## Image Processing

### `internal/image/io.go`

- **Read**: Auto-detects format by magic bytes (PNG: `\x89PNG`, JPEG: `\xFF\xD8`). Returns `image.Image` interface.
- **Write**: Saves to file in PNG (default) or JPEG (`--output-format jpeg`). Quality 90 for JPEG.
- **Auto filename**: `potaco-YYYYMMDD-HHMMSS.png` when `--output` is not specified, saved in current working directory.

### `internal/image/mask.go`

**Mask from file** (`--mask`):
- Loads PNG/grayscale image
- Converts: any non-black pixel becomes white, black stays black
- Outputs as 8-bit grayscale PNG with proper dimensions matching the source image

**Mask from `--mask-rect x,y,w,h`**:
- Creates a new image matching source dimensions
- Fills entirely black, then draws white rectangle at (x, y) with size (w, h)

**Mask from `--mask-circle x,y,r`**:
- Creates a new image matching source dimensions
- Fills entirely black, then draws filled white circle centered at (x, y) with radius r

All masks are validated: dimensions must match the source image. If they don't, the mask is scaled to match (with a warning to stderr).

### `internal/image/canvas.go`

**Outpaint pipeline** (`--extend top,bottom,left,right`):
1. Load source image, get dimensions (W, H)
2. Create new canvas sized (W + left + right, H + top + bottom)
3. Paste source image at offset (left, top) using `golang.org/x/image/draw`
4. Fill new (empty) areas with a neutral color (gray 128)
5. Generate mask: white where new pixels are, black where original image is
6. Return composite image + mask as a pair for the edit API call
7. The edit API fills the white-masked area based on the prompt

### `internal/image/view.go`

**Terminal image display** (`--view` flag):
- Detects terminal protocol support:
  - iTerm2: checks `TERM_PROGRAM=iTerm.app`
  - Kitty: checks `TERM_PROGRAM=kitty` or `TERM=xterm-kitty`
  - Sixel: checks terminal capabilities
- If supported: encodes image using the detected protocol and writes escape sequences to stdout
- If unsupported: prints "Saved to: \<path\> (terminal does not support inline image preview)"
- Does not interfere with `--json` or `--stdout` flags (those take precedence)

## Output Model

### Progressive Disclosure

**Default** (human-friendly, prints to stdout):
```
$ potaco gen --prompt "a cat"
Saved to: potaco-20260624-153201.png
```

With `--view`:
```
$ potaco gen --prompt "a cat" --view
[inline image preview]
Saved to: potaco-20260624-153201.png
```

**JSON mode** (`--json`, agent-friendly, structured JSON to stdout):
```json
{
  "path": "potaco-20260624-153201.png",
  "format": "png",
  "width": 1024,
  "height": 1024,
  "model": "dall-e-3",
  "params": {
    "size": "1024x1024",
    "quality": "standard",
    "n": 1
  },
  "latency_ms": 3420
}
```

For `--n` greater than 1, returns a JSON array of image objects.

**Stdout mode** (`--stdout`):
- Raw image bytes piped to stdout (binary)
- All other stdout output suppressed
- Can optionally also save to file with `--output`
- Useful for piping: `potaco gen --prompt "x" --stdout | other-tool`

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error (missing config, invalid values) |
| 3 | API error (HTTP errors, parse failures, rate limit exhausted) |
| 4 | Image processing error (unreadable file, invalid mask, canvas failure) |

## Error Handling

### Error Output

All errors print to stderr.

**Default** (human-readable):
```
Error: Failed to generate image (HTTP 429): Rate limited. Retried 2 times.
```

**JSON mode** (`--json` flag enables JSON errors to stderr):
```json
{"error": {"type": "api_error", "status": 429, "message": "Rate limited", "retried": 2}}
```

### Retry

- HTTP 429 and 5xx: retry with exponential backoff
- Backoff: 1s, 2s, 4s with jitter (0-500ms)
- Configurable via `--retries` or config `retries` field
- Network errors (connection refused, timeout): retried once
- 4xx errors (except 429): no retry, immediate failure
- Retry attempts logged to stderr with `--verbose`

### Dry-Run (`--dry-run`)

Validates everything without making API calls or writing files:
- Config present and complete (base_url, api_key, model)
- Prompt is non-empty
- For `edit`: `--image` file exists and is readable; mask is valid if provided
- For outpaint: extend values parse correctly
- Parameters are valid within known constraints

Prints the API request payload to stdout as JSON:
```json
{
  "method": "POST",
  "url": "https://api.openai.com/v1/images/generations",
  "headers": {"Content-Type": "application/json", "Authorization": "Bearer [REDACTED]"},
  "body": {"prompt": "a cat", "model": "dall-e-3", "size": "1024x1024"}
}
```

Exit 0 if valid, exit 1 with error if invalid.

## Testing Strategy

### Unit Tests

**`internal/config`**:
- Config file parsing (valid YAML, malformed YAML, missing file)
- Env var override precedence (flag > env > config > preset)
- Provider preset application

**`internal/provider`**:
- `httptest.Server` mock for both endpoints
- Verify request body format: JSON for generation, multipart for edit
- Verify headers (Authorization, Content-Type)
- Retry logic: simulated 429 (then 200), simulated 5xx, max retries exhausted
- Response parsing: b64_json response, URL response (fetch mocked)

**`internal/image`**:
- Mask generation: rect mask dimensions, circle mask, file loading, dimension mismatch scaling
- Canvas operations: outpaint canvas sizing, source image positioning, mask generation
- Image I/O: PNG read/write roundtrip, JPEG read/write, format detection by magic bytes

**`internal/cli`**:
- Flag parsing for each subcommand
- Output formatting: text mode, JSON mode, stdout mode
- Dry-run validation (valid and invalid cases)
- Exit code verification
- Table-driven tests for edge cases (empty prompt, missing file, invalid extend format)

### Integration Tests

- `//go:build integration` tag
- Full CLI path with `--dry-run` (no API calls needed)
- Optional: real API tests when `POTACO_API_KEY` env var is set

## Dependencies

- **github.com/spf13/cobra** — CLI framework
- **gopkg.in/yaml.v3** — Config file parsing
- **golang.org/x/image/draw** — High-quality image resizing for canvas/compositing

No CGO dependencies. Pure Go for easy cross-compilation.

## Future Considerations (Out of Scope for v0)

- Interactive ASCII mask painting
- Batch generation from a file of prompts
- Provider-specific parameter auto-detection
- Image upscaling and variation endpoints
- Progress bars for long-running generation
- Plugin system for custom providers
