# Sync Instructions for refactor Skill

## Source of truth

This skill is a Droid-native port of the upstream repository at:

https://github.com/code-yeongyu/oh-my-openagent/tree/dev/packages/shared-skills/skills/refactor

The upstream directory contains only:

- `SKILL.md` — the main skill prompt

The upstream `SKILL.md` contains a lot of omo-specific workflow text (e.g., `call_omo_agent`, `lsp_*`, `ast_grep`, `background_output`, the `REFACTOR_TEAM_MODE_ADDENDUM`, `team_*` tools, `sisyphus` subagent types, `~/.omo` paths). This local port removed that entire addendum and rewrote the workflow to use Droid-native tools.

## How to check for upstream changes

1. Fetch the upstream `SKILL.md`:

```bash
curl -L -o /tmp/refactor-upstream.md https://raw.githubusercontent.com/code-yeongyu/oh-my-openagent/dev/packages/shared-skills/skills/refactor/SKILL.md
```

2. Compare it against the local copy:

```bash
diff /home/ngct/.factory/skills/refactor/SKILL.md /tmp/refactor-upstream.md
```

3. Also check whether any new files were added upstream by opening the directory listing:

https://github.com/code-yeongyu/oh-my-openagent/tree/dev/packages/shared-skills/skills/refactor

If the upstream directory contains files other than `SKILL.md`, treat them as new supporting files and follow the "How to update" section below.

## How to update

### If only `SKILL.md` changed

1. Read the upstream `SKILL.md` and the local `SKILL.md` side by side.
2. Port the new content into the local skill:
   - Remove/replace omo-specific wording, tool names, paths, and brand references.
   - Replace omo agent tooling (`call_omo_agent`, `background_output`, `team_*`, `multi_agent_v1`, `lsp_*`, `ast_grep`) with Droid-native tools (`Task`, `Grep`, `Glob`, `Read`, `Edit`, `Execute`, `TodoWrite`).
   - The upstream `REFACTOR_TEAM_MODE_ADDENDUM` section is omo-specific and should **not** be copied verbatim. If the upstream team-mode protocol changed, decide whether the concepts are worth porting to Droid's parallel `Task` model; if not, leave the local skill without it.
   - Keep the Droid-native YAML frontmatter (`name`, `description`, optional `user-invocable`, `disable-model-invocation`).
3. Save the updated `SKILL.md`.
4. Re-run the omo-reference check.

### If new supporting files were added upstream

1. Determine if they are omo-free or omo-specific.
2. If omo-free (e.g., a generic checklist or reference table), copy them into the local directory.
3. If omo-specific (e.g., scripts that call `call_omo_agent`, configs with `~/.omo` paths, team spec JSON), port the concepts to Droid-native tooling or skip them entirely and document the gap.

## Mandatory omo-cleanliness check

After every sync, run this exact command from the skill root and verify it returns no matches:

```bash
cd /home/ngct/.factory/skills/refactor
grep -RinE '\bomo\b|oh-my-openagent|call_omo|lsp_diagnostics|lsp_diagnostic|lsp_prepare|lsp_rename|LspGoto|LspFind|LspDocument|LspWorkspace|ast_grep|ast-grep|\$omo:|lazycodex|sisyphus|background_output|multi_agent|team_create|team_send|team_task|team_delete|team_shutdown|team_list|team_mode|team_status|team_approve|REFACTOR_TEAM|REFACTOR_TEMPLATE' .
```

If the grep returns any matches, stop and remove or port those references before declaring the sync complete.

## What to do if you are unsure

- If an upstream change introduces a tool or workflow that has no Droid-native equivalent (e.g., LSP rename, AST-grep, team-mode agents), do not copy it verbatim. Instead, document the gap in this file, report to the user, and propose a Droid-native equivalent.
- If the upstream changes the phase names, triggers, or success criteria, port those changes while preserving the Droid-native tool replacements.

## Last synced

2026-06-25
