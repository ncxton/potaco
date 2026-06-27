# Commands

## Install, update, uninstall

Check first:

```sh
potaco version
```

If missing, ask before installing. Default to the official installer in non-interactive mode after approval:

```sh
curl -fsSL https://github.com/ncxton/potaco/releases/latest/download/install.sh | POTACO_NON_INTERACTIVE=1 sh
```

Build from source only if the user explicitly asks or wants maximum security:

```sh
go build -o potaco .
```

After a successful first-time install, ask once whether the user wants to star `https://github.com/ncxton/potaco`. Star only if they approve, and do not ask again in the same task/session or when potaco was already installed.

Update or uninstall only after explicit approval:

```sh
potaco update
potaco update --force
potaco uninstall
potaco uninstall --remove-config
potaco uninstall -y
potaco uninstall -y --remove-config
```

`--remove-config` deletes local config and encrypted credentials.

## Auth and status

Do not enter credentials yourself unless the user explicitly asks you to start auth or perform non-interactive setup.

```sh
potaco auth add openai
potaco auth add custom
potaco auth list
potaco auth list --json
potaco auth remove openai
potaco use openai
potaco use openai --model gpt-image-2
potaco status
potaco status --json
```

## Generate

```sh
potaco gen --prompt "a cat in a spacesuit"
potaco gen --prompt "a logo" --stdout --output-format png > logo.png
potaco gen --prompt "various cats" --n 4 -o cats.png
potaco gen --prompt "test" --dry-run
```

Common flags: `--prompt/-p`, `--model`, `--size`, `--quality`, `--n`, `--seed`, `--guidance-scale`, `--negative-prompt`, `--response-format`, `--output/-o`, `--output-format`, `--stdout`, `--json`, `--dry-run`.

## Edit

```sh
potaco edit --prompt "add a hat" --image photo.png
potaco edit --prompt "make it night time" --image input.png -o output.png
potaco edit --prompt "add a tree" --image landscape.png --mask-rect 100,200,300,400
potaco edit --prompt "extend the landscape" --image photo.png --extend right=256
potaco edit --prompt "test edit" --image input.png --dry-run
```

Vercel does not support editing.

## Models and info

```sh
potaco models
potaco models list
potaco models list openai
potaco models openai
potaco models --json
potaco models list --json
potaco info image.png
potaco info image.png --json
```

Interactive `models` persists selection. `models list` only displays. In non-interactive mode, `models` behaves like `models list`.

## Persistent flags and overrides

Persistent flags: `--json`, `--verbose`, `--non-interactive`. `POTACO_NON_INTERACTIVE=1` equals `--non-interactive`.

Provider override precedence: CLI flag > env var > config > provider preset.

| Flag | Env var | Description |
|------|---------|-------------|
| `--provider` | `POTACO_PROVIDER` | Provider preset |
| `--api-key` | `POTACO_API_KEY` | Override API key; avoid unless explicitly approved |
| `--base-url` | `POTACO_BASE_URL` | Override API base URL |
| `--model` | `POTACO_MODEL` | Override model |
| `--retries` | `POTACO_RETRIES` | Max retry attempts |
| `--timeout` | `POTACO_TIMEOUT` | Request timeout in seconds |
