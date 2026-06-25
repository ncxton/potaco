# Sync Instructions for programming Skill

## Source of truth

This skill is a Droid-native port of the upstream repository at:

https://github.com/code-yeongyu/oh-my-openagent/tree/dev/packages/shared-skills/skills/programming

The upstream directory contains:

- `SKILL.md` — the main skill prompt
- `references/` — per-language reference manuals (python/, rust/, typescript/, rust-ub/, go/)
- `scripts/` — project bootstrap and `check-no-excuse-rules` helpers for python/, rust/, typescript/, go/

## What this port already did

1. Removed all omo-specific wording, branding, and tool references (e.g., `call_omo_agent`, `lsp_*`, `ast_grep`, `multi_agent_v1`, `team_*`, `~/.omo`, `sisyphus`, `oh-my-openagent`).
2. Replaced omo tooling with Droid-native equivalents (`Task` with `subagent_type="worker"`, `Grep`, `Glob`, `Read`, `Edit`, `Execute`, `TodoWrite`).
3. Copied the `references/` and `scripts/` directories verbatim because they contained no omo-specific references.

## How to check for upstream changes

1. Fetch the upstream directory listing to see if files were added, removed, or renamed:

```bash
git ls-remote --heads https://github.com/code-yeongyu/oh-my-openagent.git dev
```

Then either:

- Option A: open https://github.com/code-yeongyu/oh-my-openagent/tree/dev/packages/shared-skills/skills/programming in a browser and compare the file tree to `~/.factory/skills/programming/`.
- Option B: sparse-checkout the upstream directory to `/tmp/omo-sync-programming/` and diff against the local copy:

```bash
cd /tmp
rm -rf omo-sync-programming
mkdir omo-sync-programming
cd omo-sync-programming
git init
git remote add origin https://github.com/code-yeongyu/oh-my-openagent.git
git config core.sparseCheckout true
echo "packages/shared-skills/skills/programming/" > .git/info/sparse-checkout
git fetch --depth 1 origin dev
git checkout dev

diff -r /home/ngct/.factory/skills/programming/ packages/shared-skills/skills/programming/
```

3. Look for three kinds of changes:

- **New files** in upstream that do not exist locally.
- **Deleted files** in upstream that still exist locally.
- **Modified files** where the upstream content differs from the local copy.

Pay special attention to `SKILL.md` and files under `references/` and `scripts/`.

## How to update

### If only `references/` or `scripts/` files changed and they are still omo-free

1. Copy the changed upstream files into the matching local directories.
2. Re-run the omo-reference check (see below) to confirm they remain clean.
3. Update this `SYNC.md` with the new sync date if desired.

### If `SKILL.md` changed

1. Read the upstream `SKILL.md` and the local `SKILL.md` side by side.
2. Port the new content into the local skill:
   - Remove/replace omo-specific wording, tool names, paths, and brand references.
   - Replace omo agent tooling (`call_omo_agent`, `background_output`, `team_*`, `multi_agent_v1`, `lsp_*`, `ast_grep`) with Droid-native tools (`Task`, `Grep`, `Glob`, `Read`, `Edit`, `Execute`, `TodoWrite`).
   - Preserve references to `references/` and `scripts/` if they remain valid.
   - Keep the Droid-native YAML frontmatter (`name`, `description`, optional `user-invocable`, `disable-model-invocation`).
3. Save the updated `SKILL.md`.
4. Re-run the omo-reference check.

### If new supporting files were added upstream

1. Determine if they are omo-free technical references/scripts or omo-specific workflow files.
2. If omo-free, copy them into the matching local directory.
3. If omo-specific (e.g., a file that references `call_omo_agent`, `lsp_diagnostics`, team mode, or `~/.omo`), port the concepts to Droid-native tooling before saving.

## Mandatory omo-cleanliness check

After every sync, run this exact command from the skill root and verify it returns no matches:

```bash
cd /home/ngct/.factory/skills/programming
grep -RinE '\bomo\b|oh-my-openagent|call_omo|lsp_diagnostics|lsp_diagnostic|lsp_prepare|lsp_rename|LspGoto|LspFind|LspDocument|LspWorkspace|ast_grep|ast-grep|\$omo:|lazycodex|sisyphus|background_output|multi_agent|team_create|team_send|team_task|team_delete|team_shutdown|team_list|team_mode|team_status|team_approve|REFACTOR_TEAM|REFACTOR_TEMPLATE' .
```

If the grep returns any matches, stop and remove or port those references before declaring the sync complete.

## What to do if you are unsure

- If an upstream change introduces a tool or workflow that has no Droid-native equivalent (e.g., LSP rename, AST-grep, team-mode agents), do not copy it verbatim. Instead, document the gap in this file, report to the user, and propose a Droid-native equivalent.
- If an upstream change is purely domain/technical guidance (e.g., a new Rust UB pattern, a new Go library recommendation), it is usually safe to port after the omo check.

## Last synced

2026-06-25
