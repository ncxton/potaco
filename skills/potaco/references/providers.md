# Provider Details

Potaco supports three built-in providers, each registered via a factory in
`internal/adapter/<provider>/init.go`. Adapters are selected by name and
created with an API key and options.

## Provider Support Matrix

| Feature | openai | fal | vercel |
|---------|--------|-----|--------|
| Generation | Yes | Yes | Yes |
| Editing | Yes | Yes | No |
| Model discovery | GET `/v1/models` | POST `api.fal.ai/v1/models` | GET `/v1/models` + endpoints |
| Auth header | `Bearer <key>` | `Key <key>` | `Bearer <key>` |

## openai

**Default base URL**: `https://api.openai.com/v1`
**Default model**: `gpt-image-2`
**Auth header**: `Bearer <api-key>`

### Endpoints

- Generate: `POST /v1/images/generations` (or `/images/generations` if base URL ends with `/v1`)
- Edit: `POST /v1/images/edits` (multipart/form-data)
- Discover: `GET /v1/models`

### Fallback models (used when API discovery fails)

| Model ID | Display Name | Edit | Capabilities |
|----------|-------------|------|--------------|
| gpt-image-2 | GPT Image 2 | Yes | size, quality, n, background, output_format, output_compression, moderation |
| gpt-image-1 | GPT Image 1 | Yes | size, quality, n |
| gpt-image-1-mini | GPT Image 1 Mini | Yes | size, quality, n |
| dall-e-3 | DALL-E 3 | No | size, quality, style, n |
| dall-e-2 | DALL-E 2 | Yes | size, quality, n |

### Model params (gpt-image-2)

| Param | Type | Default | Enum values |
|-------|------|---------|-------------|
| size | enum | 1024x1024 | 1024x1024, 1536x1024, 1024x1536, auto |
| quality | enum | auto | auto, low, medium, high |
| n | int | 1 | |
| background | enum | auto | transparent, opaque, auto |
| output_format | enum | png | png, jpeg, webp |
| output_compression | int | 0 | |
| moderation | enum | auto | auto, low |

### Model params (dall-e-3)

| Param | Type | Default | Enum values |
|-------|------|---------|-------------|
| size | enum | 1024x1024 | 1024x1024, 1792x1024, 1024x1792 |
| quality | enum | standard | standard, hd |
| style | enum | vivid | vivid, natural |
| n | int | 1 | (always 1 for dall-e-3) |

## fal

**Default base URL**: `https://fal.run`
**Discovery base URL**: `https://api.fal.ai`
**Default model**: `fal-ai/flux/dev`
**Auth header**: `Key <api-key>`

### Endpoints

- Generate: `POST https://fal.run/<model-id>` (model ID is the path, e.g. `fal-ai/flux/dev`)
- Edit: `POST https://fal.run/<model-id>/image-to-image` (JSON body, not multipart)
- Discover: `POST https://api.fal.ai/v1/models`

### Fallback models

| Model ID | Display Name | Edit | Capabilities |
|----------|-------------|------|--------------|
| fal-ai/flux/dev | Flux Dev | No | guidance_scale, num_inference_steps, seed, output_format, image_size, num_images, enable_safety_checker |
| fal-ai/flux/schnell | Flux Schnell | No | num_inference_steps, seed, output_format, image_size, num_images |
| fal-ai/nano-banana | Nano Banana | Yes | aspect_ratio, output_format, safety_tolerance, system_prompt |

### Model params (fal-ai/flux/*)

| Param | Type | Default |
|-------|------|---------|
| guidance_scale | float | 3.5 |
| num_inference_steps | int | 50 |
| seed | int | 0 |
| output_format | enum | png (png, jpeg, webp) |
| image_size | string | 1024x1024 |
| num_images | int | 1 |
| enable_safety_checker | bool | true |

### Model params (fal-ai/nano-banana)

| Param | Type | Default |
|-------|------|---------|
| aspect_ratio | string | 1:1 |
| output_format | enum | png (png, jpeg) |
| safety_tolerance | int | 2 |
| system_prompt | string | |

## vercel

**Default base URL**: `https://ai-gateway.vercel.sh/v1`
**Default model**: `openai/gpt-image-2`
**Auth header**: `Bearer <api-key>`

### Endpoints

- Generate: `POST /v1/images/generations`
- Edit: Not supported (returns `ErrEditNotSupported`)
- Discover: `GET /v1/models` + `GET /v1/models/<id>/endpoints`

### Fallback models

| Model ID | Display Name | Capabilities |
|----------|-------------|--------------|
| openai/gpt-image-2 | gpt-image-2 | size, quality, n |
| openai/dall-e-3 | dall-e-3 | size, quality, style, n |
| bfl/flux-2-pro | flux-2-pro | outputFormat, aspectRatio |

### How Vercel model IDs work

Vercel AI Gateway uses a `provider/model` naming convention. The provider
prefix (before the `/`) determines which hardcoded params apply:

- `openai/*` models get size, quality, n params
- `bfl/*` models get outputFormat, aspectRatio params

## Adding a Custom Provider

The codebase supports adding new providers by implementing the
`adapter.Adapter` interface and registering via `init()`:

1. Create `internal/adapter/<name>/<name>.go` implementing `Generate`, `Edit`,
   `DiscoverModels`, `Verify`, `ModelParams`, `Name`, `AuthHeader`.
2. Call `adapter.Register("<name>", factory)` in an `init()` function.
3. Blank-import the new package in `internal/cli/helpers.go` for side-effect
   registration.
4. Add a preset entry to `providerPresets` in `helpers.go`.

This is an advanced codebase task, not a CLI usage task.
