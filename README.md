# claudemod

A workflow orchestrator for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) that guides AI-assisted development through structured, multi-phase workflows. Instead of a single open-ended Claude session, claudemod breaks work into focused phases — discuss, spec, implement, review — so each session has clear goals and exit criteria.

## Quick Start

```bash
# Install
git clone <repo-url> && cd claudemod
go build -o bin/claudemod cmd/claudemod/main.go

# Or, if you have Task installed:
task install   # builds and copies to ~/.local/bin/claudemod

# Set up a project
cd /path/to/your/project
claudemod init

# Run a workflow
claudemod run bootstrap
```

## Prerequisites

- **Go 1.24+**
- **macOS or Linux** (no Windows support — requires PTY)
- **Claude Code CLI** installed at `~/.local/bin/claude` or anywhere in `$PATH`
- **[Task](https://taskfile.dev/)** (optional, for build shortcuts)

## Commands

| Command                    | Description                                                       |
| -------------------------- | ----------------------------------------------------------------- |
| `claudemod init`           | Scaffold `.claudemod/` and `.claude/` directories in your project |
| `claudemod run <workflow>` | Run a workflow (see below)                                        |

## Workflows

### `bootstrap` — Learn a new codebase

Explores the codebase, asks you clarifying questions, and generates structured spec files that document the project's architecture, domains, data flows, and design decisions.

Run this first on any project. The specs it generates are used by all other workflows.

```
bootstrap
```

### `explain` — Understand how something works

A guided investigation where you ask questions about the codebase and Claude traces through the actual code to answer them. Think of it as asking a senior developer to walk you through a module.

Along the way, Claude compares what the code does against what the specs say, flagging any discrepancies. At the end, specs are updated to match reality — correcting assumptions from previous bootstrap runs.

```
ask-question → deep-dive → update-specs
```

### `feature` — Build a feature with TDD

Full development lifecycle: discuss requirements, design the architecture, break work into tasks, write failing tests, implement, refactor, review, and update specs.

Each phase has exit criteria that must be met before advancing. Phases can roll back to earlier phases when problems are discovered (e.g. spec gaps found during implementation).

```
discuss-feature → spec-plan → scope-plan → tdd-red → tdd-green → tdd-refactor → design-review → code-review → synthesize-specs
```

### `bugfix` — Fix a bug with TDD

Same structured lifecycle as `feature`, but starts with bug investigation instead of feature discussion. Claude helps you reproduce the bug, theorize the root cause, verify it against code, then fix it through TDD.

```
describe-bug → spec-plan → scope-plan → tdd-red → tdd-green → tdd-refactor → design-review → code-review → synthesize-specs
```

### `task` — Structured work without TDD

For non-TDD work like refactoring, adding documentation, updating configurations, or other tasks that still benefit from structured phases but don't need the full red/green/refactor cycle. Claude discusses the task, plans the work, executes it while verifying no regressions, reviews the changes, and updates specs.

```
discuss-task → task-plan → execute-task → code-review → synthesize-specs
```

### `backlog` — Generate a prioritized backlog

Discuss a feature, draft a technical spec, break it into tasks, then transform those tasks into prioritized user stories with story points, dependencies, and acceptance criteria. Produces a `STORIES.md` backlog ready for sprint planning.

```
discuss-feature → spec-plan → scope-plan → generate-stories
```

## How It Works

When you run a workflow, claudemod spawns a Claude Code session for each phase with a system prompt tailored to that phase's goals. You interact with Claude normally — it just has focused instructions for what to accomplish.

**Phase transitions** — Claude writes a `SESSION_STATE.json` file when a phase completes. claudemod reads it and spawns the next session automatically.

**Resumable** — If you interrupt a workflow (Ctrl+C), it resumes at the interrupted phase on next run. Progress is tracked in `.claudemod/PHASE_LOG.jsonl`.

**Rollback** — When a phase discovers a concrete problem (requirements misunderstood, spec gaps, wrong test assumptions), it can roll back to an earlier phase to fix the issue before continuing.

**Multi-project** — Workflows support multi-project workspaces. Planning phases see all child projects; implementation phases are scoped to only the affected repos.

## The `.claudemod/` Folder

`claudemod init` creates this structure in your project:

```
.claudemod/
  SESSION_STATE.json       Current phase and transition action
  PHASE_LOG.jsonl          Append-only history of phase transitions
  PLAN.md                  Requirements (feature/task) or bug spec (bugfix) or investigation plan (explain)
  FIX_PLAN.md              Prioritized task list for implementation phases
  STORIES.md               Prioritized user stories with points and dependencies (backlog)
  CHANGELOG.md             Dated entries summarizing completed work and spec changes
  spec/                    Architecture and domain specs
    INDEX.md               Project overview, tech stack, domain map
    {domain}/{file}.md     Per-domain specs (data models, interfaces, flows)
  refs/                    Reference templates for spec formatting
    SPEC.md                Template for domain specs
    SPEC_INDEX.md          Template for the spec index
    BUG_SPEC.md            Template for bug specs
```

## Development

```bash
task build          # Build to bin/claudemod
task test           # go test -cover ./...
task lint           # go vet + staticcheck
task install        # Build + copy to ~/.local/bin/claudemod
task clean          # Remove build artifacts
```

Without `task`:

```bash
go build -o bin/claudemod cmd/claudemod/main.go
go test -cover ./...
go test -race ./...
go vet ./...
```
