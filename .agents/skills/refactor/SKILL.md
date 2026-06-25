---
name: refactor
description: "Intelligent refactor command. Triggers: refactor, refactoring, cleanup, restructure, extract, simplify, modernize."
---

# Intelligent Refactor

## Usage
```
/refactor  [--scope=] [--strategy=]

Arguments:
  refactoring-target: What to refactor. Can be:
   - File path: src/auth/handler.ts
   - Symbol name: "AuthService class"
   - Pattern: "all functions using deprecated API"
   - Description: "extract validation logic into separate module"

Options:
  --scope: Refactoring scope (default: module)
   - file: Single file only
   - module: Module/directory scope
   - project: Entire codebase

  --strategy: Risk tolerance (default: safe)
   - safe: Conservative, maximum test coverage required
   - aggressive: Allow broader changes with adequate coverage
```

## What This Command Does

Performs intelligent, deterministic refactoring with full codebase awareness. Unlike blind search-and-replace, this command:

1. **Understands your intent** — Analyzes what you actually want to achieve
2. **Maps the codebase** — Builds a definitive codemap before touching anything
3. **Assesses risk** — Evaluates test coverage and determines verification strategy
4. **Plans meticulously** — Creates a detailed plan
5. **Executes precisely** — Step-by-step refactoring with code search and edits
6. **Verifies constantly** — Runs tests after each change to ensure zero regression

---

# PHASE 0: INTENT GATE (MANDATORY FIRST STEP)

**BEFORE ANY ACTION, classify and validate the request.**

## Step 0.1: Parse Request Type

| Signal | Classification | Action |
|--------|----------------|--------|
| Specific file/symbol | Explicit | Proceed to codebase analysis |
| "Refactor X to Y" | Clear transformation | Proceed to codebase analysis |
| "Improve", "Clean up" | Open-ended | **MUST ask**: "What specific improvement?" |
| Ambiguous scope | Uncertain | **MUST ask**: "Which modules/files?" |
| Missing context | Incomplete | **MUST ask**: "What's the desired outcome?" |

## Step 0.2: Validate Understanding

Before proceeding, confirm:
- [ ] Target is clearly identified
- [ ] Desired outcome is understood
- [ ] Scope is defined (file/module/project)
- [ ] Success criteria can be articulated

**If ANY of above is unclear, ASK CLARIFYING QUESTION:**

```
I want to make sure I understand the refactoring goal correctly.

**What I understood**: [interpretation]
**What I'm unsure about**: [specific ambiguity]

Options I see:
1. [Option A] - [implications]
2. [Option B] - [implications]

**My recommendation**: [suggestion with reasoning]

Should I proceed with [recommendation], or would you prefer differently?
```

## Step 0.3: Create Initial Todos

**IMMEDIATELY after understanding the request, create todos:**

```
TodoWrite([
  {"content": "PHASE 1: Codebase Analysis - launch parallel explore agents", "status": "pending"},
  {"content": "PHASE 2: Build Codemap - map dependencies and impact zones", "status": "pending"},
  {"content": "PHASE 3: Test Assessment - analyze test coverage and verification strategy", "status": "pending"},
  {"content": "PHASE 4: Plan Generation - create detailed refactoring plan", "status": "pending"},
  {"content": "PHASE 5: Execute Refactoring - step-by-step with continuous verification", "status": "pending"},
  {"content": "PHASE 6: Final Verification - full test suite and regression check", "status": "pending"}
])
```

---

# PHASE 1: CODEBASE ANALYSIS (PARALLEL EXPLORATION)

**Mark phase-1 as in_progress.**

## 1.1: Launch Parallel Explore Agents (BACKGROUND)

Fire ALL of these simultaneously using the `Task` tool with `subagent_type` set to an explore-capable worker droid:

```
// Agent 1: Find the refactoring target
Task(
  subagent_type="worker",
  description="Find refactoring target",
  prompt="Find all occurrences and definitions of [TARGET].
  Report: file paths, line numbers, usage patterns."
)

// Agent 2: Find related code
Task(
  subagent_type="worker",
  description="Find related code",
  prompt="Find all code that imports, uses, or depends on [TARGET].
  Report: dependency chains, import graphs."
)

// Agent 3: Find similar patterns
Task(
  subagent_type="worker",
  description="Find similar patterns",
  prompt="Find similar code patterns to [TARGET] in the codebase.
  Report: analogous implementations, established conventions."
)

// Agent 4: Find tests
Task(
  subagent_type="worker",
  description="Find tests for target",
  prompt="Find all test files related to [TARGET].
  Report: test file paths, test case names, coverage indicators."
)

// Agent 5: Architecture context
Task(
  subagent_type="worker",
  description="Find architecture context",
  prompt="Find architectural patterns and module organization around [TARGET].
  Report: module boundaries, layer structure, design patterns in use."
)
```

## 1.2: Direct Tool Exploration (WHILE AGENTS RUN)

While background agents are running, use direct tools:

### Code Search Tools for Precise Analysis:

Use the `Grep` and `Glob` tools for searching:
- `Grep` with pattern and path to find usages across the workspace
- `Glob` with patterns to find file structures
- `Read` to inspect file contents and understand definitions

### Grep for Text Patterns:

```
Grep(pattern="[search_term]", path="src/", glob_pattern="*.ts")
```

### Glob for File Discovery:

```
Glob(patterns=["**/*.test.ts", "**/*.spec.ts"], folder="src/")
```

## 1.3: Collect Background Results

Wait for each background `Task` agent to complete and collect their results.

**Mark phase-1 as completed after all results collected.**

---

# PHASE 2: BUILD CODEMAP (DEPENDENCY MAPPING)

**Mark phase-2 as in_progress.**

## 2.1: Construct Definitive Codemap

Based on Phase 1 results, build:

```
## CODEMAP: [TARGET]

### Core Files (Direct Impact)
- `path/to/file.ts:L10-L50` - Primary definition
- `path/to/file2.ts:L25` - Key usage

### Dependency Graph
[TARGET]
├── imports from:
│   ├── module-a (types)
│   └── module-b (utils)
├── imported by:
│   ├── consumer-1.ts
│   ├── consumer-2.ts
│   └── consumer-3.ts
└── used by:
    ├── handler.ts (direct call)
    └── service.ts (dependency injection)

### Impact Zones
| Zone | Risk Level | Files Affected | Test Coverage |
|------|------------|----------------|---------------|
| Core | HIGH | 3 files | 85% covered |
| Consumers | MEDIUM | 8 files | 70% covered |
| Edge | LOW | 2 files | 50% covered |

### Established Patterns
- Pattern A: [description] - used in N places
- Pattern B: [description] - established convention
```

## 2.2: Identify Refactoring Constraints

Based on codemap:
- **MUST follow**: [existing patterns identified]
- **MUST NOT break**: [critical dependencies]
- **Safe to change**: [isolated code zones]
- **Requires migration**: [breaking changes impact]

**Mark phase-2 as completed.**

---

# PHASE 3: TEST ASSESSMENT (VERIFICATION STRATEGY)

**Mark phase-3 as in_progress.**

## 3.1: Detect Test Infrastructure

```bash
# Check for test commands
cat package.json | jq '.scripts | keys[] | select(test("test"))'

# Or for Python
ls -la pytest.ini pyproject.toml setup.cfg

# Or for Go
ls -la *_test.go
```

## 3.2: Analyze Test Coverage

Launch a synchronous agent to analyze coverage:

```
Task(
  subagent_type="worker",
  description="Analyze test coverage",
  prompt="Analyze test coverage for [TARGET]:
  1. Which test files cover this code?
  2. What test cases exist?
  3. Are there integration tests?
  4. What edge cases are tested?
  5. Estimated coverage percentage?"
)
```

## 3.3: Determine Verification Strategy

Based on test analysis:

| Coverage Level | Strategy |
|----------------|----------|
| HIGH (>80%) | Run existing tests after each step |
| MEDIUM (50-80%) | Run tests + add safety assertions |
| LOW (<50%) | **PAUSE**: Propose adding tests first |
| NONE | **BLOCK**: Refuse aggressive refactoring |

**If coverage is LOW or NONE, ask user:**

```
Test coverage for [TARGET] is [LEVEL].

**Risk Assessment**: Refactoring without adequate tests is dangerous.

Options:
1. Add tests first, then refactor (RECOMMENDED)
2. Proceed with extra caution, manual verification required
3. Abort refactoring

Which approach do you prefer?
```

## 3.4: Document Verification Plan

```
## VERIFICATION PLAN

### Test Commands
- Unit: `bun test` / `npm test` / `pytest` / etc.
- Integration: [command if exists]
- Type check: `tsc --noEmit` / `pyright` / etc.

### Verification Checkpoints
After each refactoring step:
1. Run test command → all pass
2. Type check → clean

### Regression Indicators
- [Specific test that must pass]
- [Behavior that must be preserved]
- [API contract that must not change]
```

**Mark phase-3 as completed.**

---

# PHASE 4: PLAN GENERATION

**Mark phase-4 as in_progress.**

## 4.1: Generate the Plan

Create a detailed refactoring plan based on the codemap and test coverage:

Consider:
1. Break down into atomic refactoring steps
2. Each step must be independently verifiable
3. Order steps by dependency (what must happen first)
4. Specify exact files and line ranges for each step
5. Include rollback strategy for each step
6. Define commit checkpoints

## 4.2: Review and Validate Plan

After creating the plan:

1. **Verify completeness**: All identified files addressed?
2. **Verify safety**: Each step reversible?
3. **Verify order**: Dependencies respected?
4. **Verify verification**: Test commands specified?

## 4.3: Register Detailed Todos

Convert the plan into granular todos:

```
TodoWrite([
  // Each step from the plan becomes a todo
  {"content": "Step 1: [description]", "status": "pending"},
  {"content": "Verify Step 1: run tests", "status": "pending"},
  {"content": "Step 2: [description]", "status": "pending"},
  {"content": "Verify Step 2: run tests", "status": "pending"},
  // ... continue for all steps
])
```

**Mark phase-4 as completed.**

---

# PHASE 5: EXECUTE REFACTORING (DETERMINISTIC EXECUTION)

**Mark phase-5 as in_progress.**

## 5.1: Execution Protocol

For EACH refactoring step:

### Pre-Step
1. Mark step todo as `in_progress`
2. Read current file state with the `Read` tool
3. Verify no existing errors

### Execute Step
Use appropriate tool:

**For Symbol Renames:**
Use `Grep` to find all occurrences, then `Edit` to rename each one. Use `Glob` to find files that might contain the symbol.

**For Pattern Transformations:**
Use `Grep` with regex patterns to find the pattern, then `Edit` to replace each occurrence. Preview the search results first before making edits.

**For Structural Changes:**
Use the `Edit` tool for precise changes. Use `Read` to verify the surrounding context before editing.

### Post-Step Verification (MANDATORY)

```bash
# 1. Run tests
bun test # or appropriate test command

# 2. Type check
tsc --noEmit # or appropriate type check

# 3. Lint check
biome check . # or appropriate linter
```

### Step Completion
1. If verification passes → Mark step todo as `completed`
2. If verification fails → **STOP AND FIX**

## 5.2: Failure Recovery Protocol

If ANY verification fails:

1. **STOP** immediately
2. **REVERT** the failed change using `git checkout` or `Edit` to undo
3. **DIAGNOSE** what went wrong
4. **OPTIONS**:
   - Fix the issue and retry
   - Skip this step (if optional)
   - Ask user for guidance

**NEVER proceed to next step with broken tests.**

## 5.3: Commit Checkpoints

After each logical group of changes:

```bash
git add [changed-files]
git commit -m "refactor(scope): description

[details of what was changed and why]"
```

**Mark phase-5 as completed when all refactoring steps done.**

---

# PHASE 6: FINAL VERIFICATION (REGRESSION CHECK)

**Mark phase-6 as in_progress.**

## 6.1: Full Test Suite

```bash
# Run complete test suite
bun test # or npm test, pytest, go test, etc.
```

## 6.2: Type Check

```bash
# Full type check
tsc --noEmit # or equivalent
```

## 6.3: Lint Check

```bash
# Run linter
biome check . # or eslint, ruff, golangci-lint, etc.
```

## 6.4: Build Verification (if applicable)

```bash
# Ensure build still works
bun run build # or npm run build, cargo build, go build, etc.
```

## 6.5: Final Diagnostics

Check all changed files for issues by reading them and verifying no errors remain.

## 6.6: Generate Summary

```markdown
## Refactoring Complete

### What Changed
- [List of changes made]

### Files Modified
- `path/to/file.ts` - [what changed]
- `path/to/file2.ts` - [what changed]

### Verification Results
- Tests: PASSED (X/Y passing)
- Type Check: CLEAN
- Lint: CLEAN
- Build: SUCCESS

### No Regressions Detected
All existing tests pass. No new errors introduced.
```

**Mark phase-6 as completed.**

---

# CRITICAL RULES

## NEVER DO
- Skip verification checks after changes
- Proceed with failing tests
- Make changes without understanding impact
- Use `as any`, `@ts-ignore`, `@ts-expect-error`
- Delete tests to make them pass
- Commit broken code
- Refactor without understanding existing patterns

## ALWAYS DO
- Understand before changing
- Preview search results before applying edits
- Verify after every change
- Follow existing codebase patterns
- Keep todos updated in real-time
- Commit at logical checkpoints
- Report issues immediately

## ABORT CONDITIONS
If any of these occur, **STOP and consult user**:
- Test coverage is zero for target code
- Changes would break public API
- Refactoring scope is unclear
- 3 consecutive verification failures
- User-defined constraints violated

---

# Tool Usage Philosophy

Use Droid's built-in tools intelligently:

## Search Tools
- **`Grep`**: High-performance content search with regex, file type filtering, context lines. Use for finding usages, patterns, and definitions.
- **`Glob`**: File path search with glob patterns. Use for finding files by name, extension, or directory structure.
- **`LS`**: Directory listing. Use for understanding project structure.

## Editing Tools
- **`Read`**: Read file contents before editing. Always read first.
- **`Edit`**: Find and replace text in files. Provide unique context to narrow matches.
- **`Create`**: Create new files.

## Orchestration
- **`Task`**: Dispatch parallel subagents for independent exploration tasks. Use for codebase analysis, pattern discovery, and research.
- **`TodoWrite`**: Track multi-step progress. Update in real-time.
- **`Execute`**: Run shell commands for tests, builds, lint, type checks.

## Deprecated Code & Library Migration
When you encounter deprecated methods/APIs during refactoring:
1. Use `WebSearch` or `context7` MCP to find the recommended modern alternative
2. **DO NOT auto-upgrade to latest version** unless user explicitly requests migration
3. If user requests library migration, fetch latest API docs before making changes

---

**Remember: Refactoring without tests is reckless. Refactoring without understanding is destructive. This command ensures you do neither.**

---

## Sync

This skill is a Droid-native port with a `SYNC.md` file in its root directory that documents the upstream source of truth and the sync procedure. **Do not sync unless the user explicitly asks you to.** When asked, read `SYNC.md` (located alongside this `SKILL.md`) and follow the instructions there.
