# Provider Details

Potaco supports four providers, each registered via a factory in
`internal/adapter/<provider>/`. Adapters are selected by name and created
with an API key and options. Three are built-in presets (openai, fal, vercel);
the fourth (`custom`) has no preset and requires a user-supplied base URL.

## Provider Support Matrix

| Feature | openai | fal | vercel | custom |
|---------|--------|-----|--------|--------|
| Generation | Yes | Yes | Yes | Yes |
| Editing | Yes | Yes | No | Yes |
| Model discovery | GET `/v1/models` | GET `api.fal.ai/v1/models` | GET `/v1/models` | GET `/v1/models` |
| Auth header | `Bearer <key>` | `Key <key>` | `Bearer <key>` | `Bearer <key>` |
| Requires `--base-url` | No | No | No | Yes |

## openai

**Default base URL**: `https://api.openai.com/v1`
**Auth header**: `Bearer <api-key>`

### Endpoints

- Generate: `POST /v1/images/generations` (or `/images/generations` if base URL ends with `/v1`)
- Edit: `POST /v1/images/edits` (multipart/form-data)
- Discover: `GET /v1/models`

### Model discovery

Discovery calls `GET /v1/models` and filters for known image model IDs.
Only `gpt-image-2` is recognized as an image model. All other model IDs
are filtered out.

If discovery fails or returns no known image models, the command returns an
error. There are no fallback model lists.

### Known image model IDs

| Model ID | Edit capable |
|----------|-------------|
| gpt-image-2 | Yes |

## fal

**Default base URL**: `https://fal.run`
**Discovery base URL**: `https://api.fal.ai`
**Auth header**: `Key <api-key>`

### Endpoints

- Generate: `POST https://fal.run/<model-id>` (model ID is the path, e.g. `fal-ai/flux/dev`)
- Edit: `POST https://fal.run/<model-id>/image-to-image` (JSON body, not multipart)
- Discover: `GET https://api.fal.ai/v1/models?category=image`

### Model discovery

Discovery calls `GET https://api.fal.ai/v1/models?category=image` and
returns all models from the image category. Each model's display name comes
from the `metadata.display_name` field (falls back to the model ID if empty).
All discovered models report `SupportsEdit: true`.

If discovery fails or returns no models, the command returns an error.
There are no fallback model lists.

## vercel

**Default base URL**: `https://ai-gateway.vercel.sh/v1`
**Auth header**: `Bearer <api-key>`

### Endpoints

- Generate: `POST /v1/images/generations`
- Edit: Not supported (`SupportsEdit() returns false`)
- Discover: `GET /v1/models` (filters for `type == "image"`)

### Model discovery

Discovery calls `GET /v1/models` and includes only models where
`type == "image"`. Model display names strip the `provider/` prefix
(e.g., `openai/gpt-image-2` displays as `gpt-image-2`). All Vercel models
report `SupportsEdit: false`.

### Verification

Vercel verification is two-step:
1. `GET /v1/models` to confirm the endpoint is reachable (5xx = fail).
2. `GET /v1/models/openai/gpt-image-2/endpoints` to validate the API key (401/403 = invalid).

### How Vercel model IDs work

Vercel AI Gateway uses a `provider/model` naming convention. The provider
prefix (before the `/`) determines which upstream provider handles the
request (e.g., `openai/*`, `bfl/*`).

## custom

**Default base URL**: None (user-supplied, required)
**Auth header**: `Bearer <api-key>`

The custom provider connects to any OpenAI-compatible endpoint (e.g.,
Together, Groq, local vLLM servers). It implements the same Images API
as OpenAI.

### Requirements

- `--base-url` is required (no preset exists).
- The endpoint must expose OpenAI-compatible `/v1/images/generations` and
  `/v1/images/edits` endpoints.
- The base URL is persisted to config during `auth add`, so subsequent
  commands (gen, edit, models) do not need `--base-url` again.

### Endpoints

- Generate: `POST <base>/v1/images/generations` (or `/images/generations` if base URL ends with `/v1`)
- Edit: `POST <base>/v1/images/edits` (or `/images/edits` if base URL ends with `/v1`)
- Discover: `GET <base>/v1/models` (or `/models` if base URL ends with `/v1`)

### Model discovery

Unlike openai, the custom provider does not filter for known image model IDs.
All models returned by `GET /v1/models` are included, with `SupportsGen: true`
and `SupportsEdit: true`.

### Setup

```sh
# Interactive (prompts for API key and base URL)
potaco auth add custom

# Non-interactive
potaco auth add custom --api-key <key> --base-url https://api.example.com/v1
```

After setup, the base URL is stored in config and used automatically:

```sh
potaco gen --prompt "a cat"
potaco models
potaco edit --prompt "add a hat" --image photo.png
```

## Adapter Interface

Each provider implements the `adapter.Adapter` interface:

```go
type Adapter interface {
    Name() string
    SupportsGenerate() bool
    SupportsEdit() bool
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
    Edit(ctx context.Context, req EditRequest) (*GenerateResponse, error)
    DiscoverModels(ctx context.Context) ([]Model, error)
    Verify(ctx context.Context) error
    AuthHeader(apiKey string) string
}
```

The `Model` struct returned by `DiscoverModels`:

```go
type Model struct {
    ID           string
    DisplayName  string
    SupportsGen  bool
    SupportsEdit bool
    Capabilities []string
}
```

Provider-specific parameters pass through `ExtraParams` in `GenerateRequest`
and `EditRequest`. There is no `ModelParams` method on the interface anymore.

## Adding a New Provider

The codebase supports adding new providers by implementing the
`adapter.Adapter` interface and registering via `init()`:

1. Create `internal/adapter/<name>/<name>.go` implementing `Generate`,
   `Edit`, `DiscoverModels`, `Verify`, `SupportsGenerate`, `SupportsEdit`,
   `Name`, `AuthHeader`.
2. Call `adapter.Register("<name>", factory)` in an `init()` function.
3. Blank-import the new package in `internal/cli/helpers.go` for side-effect
   registration.
4. Add a preset entry to `providerPresets` in `helpers.go` (optional for
   providers like `custom` that have no default base URL).

This is an advanced codebase task, not a CLI usage task.
