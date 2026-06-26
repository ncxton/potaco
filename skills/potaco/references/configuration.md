# Configuration

Potaco resolves provider, API key, model, base URL, retries, and timeout
through a multi-layer precedence chain. Understanding this chain is
important when debugging unexpected behavior.

## Precedence Chain

For each setting, the first non-empty value wins:

### Provider

1. `--provider` CLI flag
2. `POTACO_PROVIDER` env var
3. `active_provider` from config file (`~/.potaco/config.yaml`)
4. Error if none set

### API key

1. `--api-key` CLI flag
2. `POTACO_API_KEY` env var
3. Credential store (`~/.potaco/credentials.enc`, decrypted)
4. Error if none found

### Model

1. `--model` CLI flag
2. `POTACO_MODEL` env var
3. `active_model` from config file
4. Provider preset default model (if via `--provider` override)

### Base URL

1. `--base-url` CLI flag
2. `POTACO_BASE_URL` env var
3. Provider preset default (openai: `https://api.openai.com/v1`, fal:
   `https://fal.run`, vercel: `https://ai-gateway.vercel.sh/v1`)

### Retries

Default: 2
1. `--retries` CLI flag
2. `POTACO_RETRIES` env var
3. `retries` from provider config in config file
4. Default: 2

### Timeout

Default: 120 seconds
1. `--timeout` CLI flag (seconds as integer string, e.g. `120`)
2. `POTACO_TIMEOUT` env var (seconds as integer string)
3. `timeout` from provider config in config file (Go Duration)
4. Default: 120s

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `POTACO_API_KEY` | Override stored API key |
| `POTACO_PROVIDER` | Override active provider |
| `POTACO_MODEL` | Override active model |
| `POTACO_BASE_URL` | Override API base URL |
| `POTACO_RETRIES` | Override retry count |
| `POTACO_TIMEOUT` | Override timeout in seconds |
| `POTACO_NON_INTERACTIVE` | Set to `1` to force non-interactive mode |

`POTACO_NON_INTERACTIVE=1` is equivalent to passing `--non-interactive` and
causes all commands to skip TUI flows and use direct non-interactive paths.

## Config File

Located at `~/.potaco/config.yaml`.

### Format (YAML)

```yaml
active_provider: openai
active_model: gpt-image-2
providers:
  openai:
    model: gpt-image-2
    retries: 3
    timeout: 180s
  fal:
    model: fal-ai/flux/dev
    retries: 2
    timeout: 120s
```

### Config commands

```sh
potaco config show
```

Prints the config file path, active provider/model, and per-provider settings.

```sh
potaco config set --model gpt-image-1
potaco config set --retries 5
potaco config set --timeout 60
```

Sets values for the active provider and saves to the config file. At least
one flag must be specified; running with no flags returns a config error.

The `--timeout` flag accepts seconds as an integer string (e.g. `60`), which
is converted to a Go Duration internally.

## Credential Storage

- Credentials are encrypted with AES-256-GCM.
- Encryption key is derived from hostname + username + salt.
- Key derivation uses scrypt.
- Salt file: `~/.potaco/.salt`
- Credentials file: `~/.potaco/credentials.enc`
- Credentials are never printed in output, logs, or dry-run (shown as
  `[REDACTED]` in dry-run auth headers).

## File Paths

| File | Purpose |
|------|---------|
| `~/.potaco/config.yaml` | Multi-provider config |
| `~/.potaco/credentials.enc` | Encrypted API keys |
| `~/.potaco/.salt` | Salt for key derivation |
| `~/.potaco/debug.log` | Raw error log (append-only) |

## Debug Log

When a `UserError` occurs, the raw error (not the friendly message) is
appended to `~/.potaco/debug.log` with a timestamp and category:

```
2025-01-15T10:30:00Z [api] generate: http 401: invalid api key
```

Categories: `config`, `api`, `image`, `generic`.

Use `--verbose` to see retry attempts and debug info on stderr during
generation and editing operations.

## Provider Presets

Defined in `internal/cli/helpers.go`:

| Provider | Base URL | Default Model |
|----------|----------|---------------|
| openai | `https://api.openai.com/v1` | `gpt-image-2` |
| fal | `https://fal.run` | `fal-ai/flux/dev` |
| vercel | `https://ai-gateway.vercel.sh/v1` | `openai/gpt-image-2` |
