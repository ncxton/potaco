# Sync Instructions for remove-ai-slops Skill

## When to sync

**Only sync when the user explicitly asks you to.** Do not sync proactively, do not sync as part of routine maintenance, and do not sync because a skill activation triggered. Syncing modifies skill files and can introduce regressions; it must be a deliberate, user-initiated action.

## Source of truth

This skill is a Droid-native port of the upstream repository at:

https://github.com/code-yeongyu/oh-my-openagent/tree/dev/packages/shared-skills/skills/remove-ai-slops

The upstream directory contains only:

- `SKILL.md` — the main skill prompt

The upstream `SKILL.md` contains omo-specific workflow text (e.g., `$omo:remove-ai-slops`, `deep` category agents, `multi_agent_v1.wait_agent`, `background_output`, `lsp_diagnostics`). This local port rewrote the workflow to use Droid-native tools.

## How to check for upstream changes

> **Path convention**: `<SKILL_ROOT>` refers to the directory containing this `SYNC.md` file. This skill may live at `~/.factory/skills/remove-ai-slops` (global) or at `<project>/.agents/skills/remove-ai-slops` (project-local). Use whichever path matches your current location.

1. Fetch the upstream `SKILL.md`:

```bash
curl -L -o /tmp/remove-ai-slops-upstream.md https://raw.githubusercontent.com/code-yeongyu/oh-my-openagent/dev/packages/shared-skills/skills/remove-ai-slops/SKILL.md
```

2. Compare it against the local copy:

```bash
diff <SKILL_ROOT>/SKILL.md /tmp/remove-ai-slops-upstream.md
```

3. Also check whether any new files were added upstream by opening the directory listing:

https://github.com/code-yeongyu/oh-my-openagent/tree/dev/packages/shared-skills/skills/remove-ai-slops

If the upstream directory contains files other than `SKILL.md`, treat them as new supporting files and follow the "How to update" section below.

## How to update

### If only `SKILL.md` changed

1. Read the upstream `SKILL.md` and the local `SKILL.md` side by side.
2. Port the new content into the local skill:
   - Remove/replace omo-specific wording, tool names, paths, and brand references.
   - Replace `$omo:remove-ai-slops` with the plain skill name `remove-ai-slops` when instructing agents to load it.
   - Replace `deep` category agents with Droid's `Task` tool using `subagent_type="worker"` (or another appropriate subagent type).
   - Replace `multi_agent_v1.wait_agent` with standard `Task` completion handling.
   - Replace `background_output(task_id=...)` with collecting results directly from the dispatched `Task` agents.
   - Replace `lsp_diagnostics` with the project's type-checker (e.g., `tsc --noEmit`, `pyright`, `basedpyright`, `golangci-lint`) run via the `Execute` tool.
   - Preserve the 10 slop categories, the quality gates, the behavior-locking phase, and the batch-of-5 parallel processing pattern, adapting only the tool invocation syntax.
   - Keep the Droid-native YAML frontmatter (`name`, `description`, optional `user-invocable`, `disable-model-invocation`).
3. Save the updated `SKILL.md`.
4. Re-run the omo-reference check.

### If new supporting files were added upstream

1. Determine if they are omo-free or omo-specific.
2. If omo-free (e.g., a generic checklist or reference table), copy them into the local directory under `<SKILL_ROOT>/`.
3. If omo-specific (e.g., scripts that call `call_omo_agent`, configs with `~/.omo` paths, agent spec files), port the concepts to Droid-native tooling or skip them entirely and document the gap.

## Mandatory omo-cleanliness check

After every sync, run this exact command from the skill root and verify it returns no matches:

```bash
cd <SKILL_ROOT>
grep -RinE '\bomo\b|oh-my-openagent|call_omo|lsp_diagnostics|lsp_diagnostic|lsp_prepare|lsp_rename|LspGoto|LspFind|LspDocument|LspWorkspace|ast_grep|ast-grep|\$omo:|lazycodex|sisyphus|background_output|multi_agent|team_create|team_send|team_task|team_delete|team_shutdown|team_list|team_mode|team_status|team_approve|REFACTOR_TEAM|REFACTOR_TEMPLATE' .
```

If the grep returns any matches, stop and remove or port those references before declaring the sync complete.

## What to do if you are unsure

- If an upstream change introduces a tool or workflow that has no Droid-native equivalent (e.g., `deep` category agents, mailbox polling, `team_*` tools), do not copy it verbatim. Instead, document the gap in this file, report to the user, and propose a Droid-native equivalent.
- If the upstream changes the slop categories, quality gates, or output format, port those changes while preserving the Droid-native tool replacements.
- If the upstream adds a new category (e.g., category 11), add it to the local `SKILL.md` and keep the numbering consistent.

## Last synced

2026-06-25
