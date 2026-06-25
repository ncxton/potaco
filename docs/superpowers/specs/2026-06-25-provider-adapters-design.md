# Provider Adapters Migration Design Spec

## Overview

Potaco migrates from flat provider presets to an interface-based adapter system, enabling per-provider API shapes, credential management, model discovery, and interactive TUI flows. Three providers are supported in v0: OpenAI, fal, and Vercel (Vercel AI Gateway).

The migration also introduces the Charm ecosystem (Bubbletea, Lipgloss, Bubbles, Huh) for interactive terminal UI flows, while keeping Cobra as the command router and ensuring full non-interactive flag-based operation for agents and CI.

## Technology

- **Language**: Go 1.26, pure Go (no CGO)
- **CLI framework**: Cobra (command routing)
- **TUI framework**: Bubbletea + Bubbles + Lipgloss + Huh (interactive flows only)
- **Image processing**: Go standard library `image` package, `golang.org/x/image/draw`
- **Config format**: YAML (`~/.potaco/config.yaml`) for non-sensitive settings
- **Credential storage**: AES-256-GCM encrypted file (`~/.potaco/credentials.enc`) with machine-derived key
- **Encryption**: Pure Go `crypto/aes` + `crypto/cipher` (no system keyring dependency)

## Architecture

### Package Layout

```
potaco/
  go.mod
  main.go
  internal/
    adapter/
      adapter.go              # Adapter interface, shared types, errors
      registry.go             # Provider registry: Get(name), List(), Register()
      openai/
        openai.go             # OpenAI adapter implementation
        models.go             # Hardcoded model parameter defaults per OpenAI model
        openai_test.go
        models_test.go
      fal/
        fal.go                # fal adapter implementation
        models.go             # Hardcoded model parameter defaults per fal model
        fal_test.go
        models_test.go
      vercel/
        vercel.go             # Vercel AI Gateway adapter implementation
        models.go             # Hardcoded model parameter defaults
        vercel_test.go
        models_test.go
    credential/
      store.go                # Encrypted credential store (read/write/encrypt/decrypt)
      encrypt.go             # AES-GCM encryption with machine-derived key
      types.go                # ProviderCredential, CredentialStore structs
      store_test.go
      encrypt_test.go
    auth/
      auth.go                 # Credential management logic (add, remove, list, get)
      types.go
      auth_test.go
    cli/
      root.go                 # root command, global flags (--json, --verbose, --non-interactive)
      gen.go                  # gen subcommand (updated to use adapter)
      edit.go                 # edit subcommand (updated to use adapter)
      edit_mask.go            # mask/outpaint helpers (mostly unchanged)
      config_cmd.go           # config show/set (simplified)
      info.go                 # image metadata (unchanged)
      helpers.go              # shared CLI helpers (updated)
      output.go               # output formatting (unchanged)
      errors.go               # exit codes (updated with new error types)
      auth_cmd.go             # auth add, auth remove/rm, auth list/ls
      use_cmd.go              # potaco use (interactive + non-interactive)
      status_cmd.go           # potaco status
      models_cmd.go           # potaco models (list + params)
    tui/
      tui.go                  # Shared TUI helpers (isTTY, launch helper, error display)
      auth_add.go             # auth add interactive flow (bubbletea model)
      use_picker.go           # use interactive picker (bubbletea model)
      model_list.go            # models interactive list (bubbletea model)
      components/
        spinner.go            # Reusable spinner wrapper
        list.go               # Reusable styled list wrapper
        confirm.go            # Reusable confirm prompt
        key_input.go          # Masked text input for API keys
    config/
      config.go               # Config file loading (updated for multi-provider format)
      types.go                # Updated config types
      config_test.go
    image/
      ... (unchanged)
```

### Dependency Flow

```
cli (top)       -> adapter, credential, auth, config, image, tui
tui             -> adapter, auth, credential, config (for business logic calls)
adapter         -> config (for types only)
auth            -> credential, config
credential      -> (no internal deps)
config          -> (no internal deps)
image           -> (no internal deps)
```

The `provider` package is replaced by `internal/adapter/` with sub-packages per provider. The `tui` package depends on `adapter` and `auth` for the actual work but contains no business logic itself.

## Adapter Architecture

### Adapter Interface

```go
package adapter

type Adapter interface {
    Name() string
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
    Edit(ctx context.Context, req EditRequest) (*GenerateResponse, error)
    DiscoverModels(ctx context.Context) ([]Model, error)
    Verify(ctx context.Context) error
    ModelParams(ctx context.Context, modelID string) ([]Param, error)
    AuthHeader(apiKey string) string
}
```

### Shared Types

```go
type Model struct {
    ID           string
    DisplayName  string
    SupportsGen  bool
    SupportsEdit bool
    Capabilities []string // e.g., "guidance_scale", "negative_prompt", "seed"
}

type Param struct {
    Name        string
    Type        string // "string", "int", "float", "bool", "enum"
    Description string
    Default     string
    EnumValues  []string
    Required    bool
}

type GenerateRequest struct {
    Prompt         string
    Model          string
    N              int
    Size           string
    Quality        string
    Style          string
    ResponseFormat string
    Seed           int
    GuidanceScale  float64
    NegativePrompt string
    ExtraParams   map[string]any // provider-specific passthrough
}

type EditRequest struct {
    Prompt         string
    Model          string
    N              int
    Size           string
    ResponseFormat string
    ImagePath      string
    MaskPath       string
    User           string
    ExtraParams   map[string]any
}

type GenerateResponse struct {
    Created int64       `json:"created"`
    Data    []ImageData `json:"data"`
}

type ImageData struct {
    B64JSON       string `json:"b64_json,omitempty"`
    URL           string `json:"url,omitempty"`
    RevisedPrompt string `json:"revised_prompt,omitempty"`
}
```

### Adapter Errors

```go
var ErrEditNotSupported = errors.New("image editing not supported by this provider")
var ErrModelNotFound = errors.New("model not found")
var ErrVerificationFailed = errors.New("provider verification failed")
var ErrDiscoveryFailed = errors.New("model discovery failed")
```

### Registry

Adapters are registered at init time. The CLI uses `adapter.Get("openai")` to obtain the adapter for a given provider name.

```go
func Get(name string) (Adapter, error)
func List() []string // returns registered provider names: ["openai", "fal", "vercel"]
```

## Provider Adapter Implementations

### OpenAI Adapter (`internal/adapter/openai/`)

**Base URL**: `https://api.openai.com/v1` (overridable via `--base-url` / `POTACO_BASE_URL`)

**Auth**: `Authorization: Bearer <key>`

**Generate**: `POST /v1/images/generations` with JSON body. Maps `GenerateRequest` fields to OpenAI request schema. Provider-specific fields (`background`, `output_format`, `output_compression`, `moderation`) pass through `ExtraParams`.

**Edit**: `POST /v1/images/edits` with multipart form data. Handles image + mask file upload, prompt, model, size, etc. Same as current implementation.

**DiscoverModels**: `GET /v1/models`, filters for known image model IDs (`gpt-image-2`, `gpt-image-1`, `gpt-image-1-mini`, `dall-e-3`, `dall-e-2`). Sets `SupportsEdit` for gpt-image models and dall-e-2. Falls back to hardcoded list on API failure, prints warning to stderr.

**Verify**: `GET /v1/models`, checks HTTP 200.

**ModelParams**: OpenAI does not expose a per-model parameter schema via API. Falls back to hardcoded defaults in `models.go`:
- `gpt-image-2`: size (1024x1024, 1536x1024, 1024x1536, auto), quality (auto/low/medium/high), n, background, output_format, output_compression, moderation
- `gpt-image-1`: size (1024x1024, 1536x1024, 1024x1536), quality (auto/low/medium/high), n
- `dall-e-3`: size (1024x1024, 1792x1024, 1024x1792), quality (standard/hd), style (vivid/natural), n (always 1)
- `dall-e-2`: size (256x256, 512x512, 1024x1024), quality (standard), n (1-10)

### fal Adapter (`internal/adapter/fal/`)

**Base URL**: `https://fal.run` (inference), `https://api.fal.ai` (discovery/verify)

**Auth**: `Authorization: Key <key>` (note: `Key` not `Bearer`)

**Generate**: `POST /<endpoint_id>` with JSON body. The `endpoint_id` is the model ID (e.g., `fal-ai/flux/dev`). Request body uses fal's schema: `prompt`, `image_size` (string enum or `{width, height}` object), `num_images` (not `n`), `seed`, `num_inference_steps`, `guidance_scale`, `output_format`, `enable_safety_checker`, `sync_mode`. The adapter maps the normalized `GenerateRequest` to fal's schema: `N` -> `num_images`, `Size` -> `image_size`, etc. Provider-specific params pass through `ExtraParams`.

Response mapping: fal returns `images[]` with `url` and dimensions, not `data[]` with `b64_json`. The adapter normalizes: if `sync_mode=true`, images may come as data URIs (base64); otherwise URLs are fetched and converted to base64 to match the shared `GenerateResponse` shape.

**Edit**: `POST /<edit_endpoint_id>` with JSON body. Edit endpoints are separate models (e.g., `fal-ai/flux/dev/image-to-image`). The adapter routes edit requests to the edit variant by appending `/image-to-image` to the model ID (or looking up a mapping in `models.go`). Request body includes `image_url` (base64 data URI of the source image), `prompt`, `strength`, `num_inference_steps`, etc. Mask sent as `mask_url` if provided (base64 data URI).

**DiscoverModels**: `GET /v1/models?category=image` on `https://api.fal.ai`. Iterates `models[]`, maps `endpoint_id` -> `Model.ID`, `metadata.display_name` -> `Model.DisplayName`. Handles pagination via `next_cursor`. Sets `SupportsEdit` for endpoints containing "image-to-image" or "edit" in the ID.

**Verify**: `GET /v1/models` on `https://api.fal.ai` with `Authorization: Key <key>`, checks HTTP 200.

**ModelParams**: fal does not expose a per-model parameter schema. Falls back to hardcoded defaults per known model family in `models.go`:
- flux models (`fal-ai/flux/*`): guidance_scale, num_inference_steps, seed, output_format, image_size, num_images, enable_safety_checker
- nano-banana models (`fal-ai/nano-banana-*`): aspect_ratio, output_format, safety_tolerance, system_prompt

### Vercel Adapter (`internal/adapter/vercel/`)

**Base URL**: `https://ai-gateway.vercel.sh/v1`

**Auth**: `Authorization: Bearer <key>`

**Generate**: `POST /v1/images/generations` with JSON body (OpenAI-compatible). Model IDs are provider-prefixed (e.g., `openai/gpt-image-2`, `bfl/flux-2-pro`). Provider-specific options go in a `providerOptions` field within the request body (e.g., `googleVertex.aspectRatio`, `blackForestLabs.outputFormat`). These pass through `ExtraParams` with a `providerOptions` key.

**Edit**: Returns `ErrEditNotSupported`. The CLI catches this and shows: "Image editing is not supported by the Vercel AI Gateway provider. Use 'potaco use openai' or 'potaco use fal' to switch to a provider that supports editing."

**DiscoverModels**: `GET /v1/models`, filters for `type == "image"` in model metadata. Model IDs are prefixed, so `DisplayName` strips the provider prefix for readability. Optionally calls `GET /v1/models/{creator}/{model}/endpoints` to enrich with `supported_parameters`.

**Verify**: Two-step check. First, `GET /v1/models` (no auth needed) to confirm the gateway is reachable. Second, to validate the API key, calls `GET /v1/models/openai/gpt-image-2/endpoints` with the Bearer key. A 401/403 means the key is invalid; any other response (200, 404 for unknown model, etc.) means the key is valid and the gateway is reachable. This works during `auth add` before a model has been selected because it uses a well-known model ID purely for key validation, not for model selection.

**ModelParams**: Attempts to read `supported_parameters` from the `/endpoints` API for the model. Falls back to hardcoded defaults based on the provider prefix (e.g., `openai/gpt-image-2` uses OpenAI's gpt-image param defaults, `bfl/flux-2-pro` uses BFL's param defaults).

### Adapter Summary

| Feature | OpenAI | fal | Vercel |
|---------|--------|-----|--------|
| Gen endpoint | `/v1/images/generations` | `/<model_id>` | `/v1/images/generations` |
| Edit endpoint | `/v1/images/edits` | `/<model_id>/image-to-image` | Not supported |
| Content-Type (gen) | `application/json` | `application/json` | `application/json` |
| Content-Type (edit) | `multipart/form-data` | `application/json` | N/A |
| Auth header | `Bearer` | `Key` | `Bearer` |
| Model discovery | `GET /v1/models` | `GET /v1/models?category=image` | `GET /v1/models` (filter type=image) |
| Discovery base URL | same as inference | `api.fal.ai` (different) | same as inference |
| Image response | `data[].b64_json` or `data[].url` | `images[].url` or data URI | `data[].b64_json` or `data[].url` |
| Param schema API | No (hardcoded) | No (hardcoded) | Yes (`/endpoints`), fallback hardcoded |

## Credential Storage & Auth System

### Encryption Design

**Machine-derived key** (`internal/credential/encrypt.go`):
- Key derived from `hostname + username + random salt` using SHA-256 as a KDF
- Salt is a 32-byte random value generated on first run, stored at `~/.potaco/.salt` with `0600` perms
- The derived 32-byte key is used with AES-256-GCM to encrypt/decrypt the credentials blob
- No passphrase prompt, seamless operation on the same machine
- Moving to another machine = re-auth required (salt differs, key differs, file is undecryptable)

**Credential file** (`~/.potaco/credentials.enc`):
```go
type CredentialStore struct {
    Providers map[string]ProviderCredential `json:"providers"`
}

type ProviderCredential struct {
    APIKey  string    `json:"api_key"`
    AddedAt time.Time `json:"added_at"`
}
```

Serialized as JSON, encrypted with AES-GCM, written to disk with `0600` perms.

### Config File Format (`~/.potaco/config.yaml`)

The YAML config no longer stores API keys. It stores non-sensitive settings:

```yaml
active_provider: openai
active_model: gpt-image-2
providers:
  openai:
    model: gpt-image-2
    retries: 2
    timeout: 120s
  fal:
    model: fal-ai/flux/dev
    retries: 2
    timeout: 120s
  vercel:
    model: openai/gpt-image-2
    retries: 2
    timeout: 120s
```

`active_provider` + `active_model` replace the flat `default` section. Each provider entry holds per-provider settings (model, retries, timeout). The API key lives only in the encrypted credential store.

### Auth Commands

**`potaco auth add <provider>`**:
- Interactive (TTY): Bubbletea TUI prompts for API key (masked input via `textinput` component), verifies connectivity by calling the adapter's `Verify()`, discovers available models, shows a `list` component for model selection
- Non-interactive (flag or non-TTY): `--api-key <key>` flag or `POTACO_API_KEY` env, verifies, assigns default model from hardcoded defaults
- If verification fails (provider unreachable): shows error and asks "Provider appears unreachable. Add anyway? [y/n]" (TUI) or `--force` flag (non-interactive)
- On success: stores key in credential store, writes provider entry to config with default model, sets as active provider

**`potaco auth remove <provider>` / `potaco auth rm <provider>`**:
- Removes the provider's key from credential store and its entry from config
- If it was the active provider, clears `active_provider` and suggests `potaco use` to select another

**`potaco auth list` / `potaco auth ls`**:
- Lists all connected providers with: provider name, model, key status (redacted, "configured" vs "missing"), added date, active marker
- TTY: formatted table with `lipgloss` styling
- `--json`: structured JSON array

### `potaco use` Command

- No args + TTY: Bubbletea interactive picker showing connected providers and their models in a nested `list` (provider -> model), sets active provider + model on selection
- `potaco use <provider>`: non-interactive shortcut, activates provider with its stored default model
- `potaco use <provider> --model <model>`: activates specific model for that provider
- No args + no TTY: error "Specify a provider: potaco use <provider>"

### Credential Access Flow

When `gen` or `edit` runs:
1. Read `active_provider` from config (or override with `POTACO_PROVIDER` env)
2. Get API key from credential store: `credentialStore.Get(providerName)` (or override with `POTACO_API_KEY` env / `--api-key` flag)
3. Build adapter: `adapter.Get(providerName)` with the key
4. Read `active_model` from config (or override with `--model` flag)
5. Proceed with the API call

## Model Discovery & Provider Verification

### Model Discovery

Each adapter implements `DiscoverModels(ctx) ([]Model, error)` returning a normalized list of image-generation-capable models.

**OpenAI**: `GET /v1/models`, filters for image-capable model IDs. Falls back to hardcoded list on failure.

**fal**: `GET /v1/models?category=image` on `api.fal.ai`. Handles pagination. Detects edit-capable models by endpoint ID pattern.

**Vercel**: `GET /v1/models`, filters for `type == "image"`. Strips provider prefix for display. Optionally enriches with `/endpoints` data.

**Fallback behavior**: If discovery fails entirely, the user sees the error and is prompted to enter a model ID manually (TUI text input, or `--model` flag non-interactively).

### Provider Verification

Each adapter implements `Verify(ctx) error`:
- **OpenAI**: `GET /v1/models` with Bearer key, checks HTTP 200
- **fal**: `GET /v1/models` with `Key` auth, checks HTTP 200
- **Vercel**: `GET /v1/models` (reachability, no auth) + `GET /v1/models/openai/gpt-image-2/endpoints` with Bearer key (key validation), checks HTTP 200 or non-auth-error

**Verification error handling**:
- 401/403: "Invalid API key for <provider>. Check your key and try again."
- 429: "Rate limited by <provider>. You can still proceed but verification was skipped."
- Network timeout: "Could not reach <provider> at <url>. Add anyway? [y/n]"
- Other errors: show the error, ask if user wants to proceed

In non-interactive mode, verification failures abort unless `--force` is passed.

### `potaco models` Command

**`potaco models`** -- List available models for the active provider:
- Interactive (TTY): Bubbletea `list` showing models with gen/edit support badges, keyboard-navigable
- Non-interactive: text table, `--json` for structured output

**`potaco models --params <model>`** -- Show supported parameters:
- Calls adapter's `ModelParams(ctx, modelID)` (dynamic discovery first, hardcoded fallback)
- Output: table of param name, type, default, description, enum values
- `--json`: structured JSON array

**`potaco models <provider>`** -- List models for a specific provider (not necessarily active):
- Useful for exploring before `auth add`

## Status Command

### `potaco status`

Shows current state of the tool. Non-interactive by default, `--json` for structured output.

**Text output**:
```
Active provider: openai
Active model:    gpt-image-2
Config file:     ~/.potaco/config.yaml
Credentials:     ~/.potaco/credentials.enc

Connected providers:
  openai    gpt-image-2          key: configured    added: 2026-06-24
  fal       fal-ai/flux/dev      key: configured    added: 2026-06-25
  vercel    openai/gpt-image-2   key: missing       added: 2026-06-25

Connection: openai reachable, fal reachable, vercel unreachable
```

**JSON output**: structured object with `active_provider`, `active_model`, `config_path`, `providers[]`, `connection_status` map.

TTY mode uses `lipgloss` styling (colored headers, status badges, connection indicators). No Bubbletea interaction needed.

## CLI Flag Changes

### Removed Flags
- `--provider` on gen/edit -- removed; use `potaco use` to switch providers
- `config set --provider <name>` -- removed; replaced by `auth add`
- `config set --base-url` / `--api-key` -- removed; base URL comes from adapter, API key from credential store

### Per-Call Override Flags (agent-friendly, preserved)
- `--api-key <key>` -- overrides credential store key for this single call
- `--base-url <url>` -- overrides adapter base URL (useful for self-hosted OpenAI-compatible proxies)
- `--model <model>` -- overrides active model for this single call

### New Global Flags
- `--non-interactive` -- forces non-interactive mode (skip all Bubbletea TUI, use flag/env fallbacks)

### Environment Variables
Existing vars remain for backward compatibility:
- `POTACO_API_KEY` -- overrides credential store for active provider
- `POTACO_MODEL` -- overrides active model
- `POTACO_RETRIES` -- overrides retries
- `POTACO_TIMEOUT` -- overrides timeout
- `POTACO_BASE_URL` -- overrides adapter base URL
- **New**: `POTACO_PROVIDER` -- overrides active provider for this invocation only (does not persist to config)

### `config` Command Changes
- `potaco config show` -- shows current config (no API keys)
- `potaco config set --retries 3 --timeout 120s` -- sets defaults for active provider
- `potaco config set --provider <provider> --model <m>` -- sets the model for a specific provider in config (uses `--provider` flag, not positional arg, to match existing `config set` flag-based pattern)
- `config list-providers` -- removed, replaced by `auth list` and `models <provider>`

## Bubbletea TUI Integration

### Charm Ecosystem Dependencies

```
github.com/charmbracelet/bubbletea      # TUI framework
github.com/charmbracelet/bubbles        # Pre-built components (list, textinput, spinner)
github.com/charmbracelet/lipgloss       # Styling/layout
github.com/charmbracelet/huh            # Interactive forms
```

### Interactive Flows

Six interactive screens, each launched only when a TTY is detected and `--non-interactive` is not set:

**1. `auth add <provider>` flow**:
- Step 1: `huh` form with masked `textinput` for API key
- Step 2: `spinner` while `Verify()` runs
- Step 3: If verification fails, `huh` confirm prompt ("Add anyway?")
- Step 4: `spinner` while `DiscoverModels()` runs
- Step 5: `bubbles/list` showing discovered models with gen/edit badges
- Step 6: On model selection, store credential + write config, show styled success

**2. `use` flow (no args)**:
- `bubbles/list` showing connected providers, expandable to show models
- Two-level selection: pick provider, then pick model
- On confirm, writes `active_provider` + `active_model` to config

**3. `models` flow (no args)**:
- `bubbles/list` of available models for active provider
- Each item: model name, gen/edit badges, description
- `--params <model>`: switches to parameter detail view (styled table)

**4. `status` output**: styled with `lipgloss` (colored headers, provider badges, connection status: green=reachable, red=unreachable, yellow=unknown)

**5. `auth list` output**: styled table with `lipgloss`

**6. Error display in interactive mode**: `lipgloss` styled error messages

### Non-Interactive Fallbacks

| Command | Interactive (TTY) | Non-Interactive |
|---------|-------------------|-----------------|
| `auth add <provider>` | TUI: key input, verify, model picker | `--api-key`, auto-verify, default model |
| `auth remove <provider>` | Confirm prompt | Proceeds immediately |
| `auth list` | Styled table | Plain text table |
| `use` (no args) | Interactive picker | Error: "Specify a provider" |
| `use <provider>` | No TUI needed | Direct switch |
| `models` | Interactive list | Plain text table |
| `models --params <m>` | Interactive param view | Plain text table |
| `status` | Styled output | Plain text output |

### TUI Code Organization

```
internal/tui/
  tui.go              # Shared helpers (isTTY, launch, error display)
  auth_add.go         # auth add interactive flow (bubbletea model)
  use_picker.go       # use interactive picker (bubbletea model)
  model_list.go       # models interactive list (bubbletea model)
  components/
    spinner.go         # Reusable spinner wrapper
    list.go            # Reusable styled list wrapper
    confirm.go         # Reusable confirm prompt
    key_input.go       # Masked text input for API keys
```

Each TUI model is a self-contained `bubbletea.Model`. The CLI commands check `tui.IsInteractive()` and either launch the TUI or fall through to the non-interactive path. The TUI models call back into `adapter` and `auth` packages for business logic; no business logic in the TUI layer.

### Cobra Integration Pattern

```go
func runAuthAdd(cmd *cobra.Command, args []string) error {
    providerName := args[0]
    if tui.IsInteractive() && !flagBool(cmd, "non-interactive") {
        return tui.RunAuthAdd(providerName)
    }
    return runAuthAddNonInteractive(cmd, providerName)
}
```

## Error Handling

### Exit Codes (preserved from v0)

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Configuration error (no active provider, missing credentials, malformed config, missing required flag in non-interactive) |
| 3 | API error (provider unreachable, auth failed, rate limit, API error, edit not supported) |
| 4 | Image processing error (invalid image, mask failure, write failure) |

### New Error Scenarios

- No active provider and no `POTACO_PROVIDER` override: config error, exit 2, "No active provider. Use 'potaco auth add <provider>' to connect one."
- Active provider has no stored key and no `POTACO_API_KEY`/`--api-key` override: config error, exit 2, "No API key for <provider>. Use 'potaco auth add <provider>' to set one."
- Provider verification fails during `auth add`: API error, exit 3, with verification detail, unless `--force`.
- Edit on Vercel: API error, exit 3, "Image editing is not supported by the Vercel AI Gateway provider."

## Testing Strategy

### Unit Tests

**`internal/adapter/openai/`**: httptest mocks for `/v1/images/generations`, `/v1/images/edits`, `/v1/models`. Verify request body, headers (Bearer auth), response parsing. Model discovery filtering and fallback. Verify HTTP 200/401 paths. Hardcoded model params per model.

**`internal/adapter/fal/`**: httptest mocks for `POST /<endpoint_id>` (verify `num_images` not `n`, `image_size` mapping, `Key` auth header). Edit via `/<endpoint_id>/image-to-image` (verify base64 image encoding, `strength` field). DiscoverModels with pagination, `SupportsEdit` detection. Response normalization (`images[].url` -> shared `GenerateResponse`).

**`internal/adapter/vercel/`**: httptest mocks for `/v1/images/generations` (verify provider-prefixed model ID, `providerOptions` passthrough). Verify `ErrEditNotSupported`. DiscoverModels `type=="image"` filtering, prefix stripping. ModelParams from `/endpoints` API with hardcoded fallback.

**`internal/credential/`**: Encrypt/decrypt roundtrip. Machine key determinism for same hostname+username+salt. Different salt produces different key. Missing salt triggers generation. Corrupted file produces clear error.

**`internal/auth/`**: Add credential (verify store + config). Remove credential (verify both stores). List (verify all providers with correct key status). Get (verify key retrieval for active provider).

**`internal/tui/`**: TUI models tested by calling `Update` and `View` with simulated messages. `IsInteractive()` with mock TTY detection.

### CLI Tests

Table-driven, `rootCmd.SetArgs` + `rootCmd.Execute()`:
- `auth add` non-interactive: `--api-key` path, verify credential store and config
- `auth remove`: verify removal from both stores
- `auth list`: text and JSON output
- `use <provider>`: verify active provider/model in config
- `status`: output shows active provider, connected providers, connection status
- `models`: model list per provider, text and JSON
- `models --params`: parameter table output
- `gen` with new config: reads active provider + model, gets key from credential store
- `gen --api-key` override: uses override key
- `gen --non-interactive`: skips TUI, uses flags
- Error cases: no active provider, no key, edit on Vercel, verification failure

### Image Package Tests

Unchanged from current implementation (mask generation, canvas, I/O, view).

## Migration Strategy

The migration is phased to keep the project in a working state at each step.

### Phase 1: Adapter Foundation
- Create `internal/adapter/` package with `Adapter` interface, shared types, registry
- Implement OpenAI adapter (extract from existing `provider` package)
- Migrate `gen` and `edit` to use the adapter interface
- Replace `internal/provider/` with `internal/adapter/`
- All existing tests pass (adapted to new package paths)

### Phase 2: Credential System
- Create `internal/credential/` with encrypted store
- Create `internal/auth/` for credential management
- Update config types to multi-provider format
- Migrate `config` command (remove old flags)
- New `auth add`, `auth remove/rm`, `auth list/ls` (non-interactive only)
- `gen` and `edit` read from credential store + new config format

### Phase 3: fal & Vercel Adapters
- Implement fal adapter with fal-specific API shape
- Implement Vercel adapter (OpenAI-compatible gen, edit-not-supported)
- Model discovery for all three adapters
- Provider verification for all three adapters

### Phase 4: Model & Status Commands
- `potaco models` command (list + params)
- `potaco status` command
- `potaco use` command (non-interactive first)

### Phase 5: Bubbletea TUI
- Add Charm dependencies to `go.mod`
- Create `internal/tui/` package
- Implement interactive flows: auth add, use picker, model list, auth list styling, status styling
- `--non-interactive` global flag
- TTY detection and flow routing

### Phase 6: Polish & Cleanup
- Remove dead code from old `presets.go`, old config format
- Update `README.md` with new command surface
- Full test suite green
- `gofmt`, `go vet` clean

Each phase produces a commitable, testable state. Phases 1-4 are non-interactive and fully testable with existing Cobra test patterns. Phase 5 adds the TUI layer on top of the working non-interactive base.

## Backward Compatibility

- `POTACO_API_KEY`, `POTACO_MODEL`, `POTACO_RETRIES`, `POTACO_TIMEOUT`, `POTACO_BASE_URL` env vars all still work
- `--api-key`, `--base-url`, `--model` flags still work on gen/edit
- Old `~/.potaco/config.yaml` format: on first run of new version, if old format detected, a migration warning guides the user to run `potaco auth add`
- `config show` still works
- `config set --provider` is removed

## Dependencies

Existing:
- `github.com/spf13/cobra` -- CLI framework
- `github.com/spf13/pflag` -- Flag parsing
- `gopkg.in/yaml.v3` -- Config file parsing
- `golang.org/x/image` -- Image processing

New:
- `github.com/charmbracelet/bubbletea` -- TUI framework
- `github.com/charmbracelet/bubbles` -- Pre-built TUI components
- `github.com/charmbracelet/lipgloss` -- Styling/layout
- `github.com/charmbracelet/huh` -- Interactive forms

No CGO dependencies. Pure Go for easy cross-compilation.
