# claudemod

A developer-focused workflow orchestrator for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) that drives incremental feature delivery through structured, multi-phase development workflows.

## How it works

claudemod runs predefined development workflows by spawning sequential Claude Code sessions, each with a phase-specific system prompt. A workflow progresses through phases (e.g. discuss, spec, scope, TDD red/green, code review) with session state persisted between phases so Claude picks up where it left off.

```
claudemod run feature
  │
  ├─ Phase 1: discuss-feature  → spawns Claude Code session with discuss prompt
  ├─ Phase 2: spec-feature     → spawns Claude Code session with spec prompt
  ├─ Phase 3: scope-feature    → spawns Claude Code session with scoping prompt
  ├─ Phase 4: tdd-red          → spawns Claude Code session with test-writing prompt
  ├─ Phase 5: tdd-green        → spawns Claude Code session with implementation prompt
  ├─ Phase 6: code-review      → spawns Claude Code session with review prompt
  └─ Phase 7: synthesize-specs → spawns Claude Code session with synthesis prompt
```

Under the hood, each session runs on a PTY bridge that sits between your terminal and the Claude Code process:

```
User Terminal <──stdin/stdout──> claudemod (PTY master) <──PTY slave──> Claude Code
```

Key behaviors:

- **Workflow-driven** — each phase has its own system prompt, exit criteria, and rollback targets. Phase transitions are automatic.
- **Session state** — state is persisted in `.claudemod/SESSION_STATE.json` between phases. If interrupted (Ctrl+C), the workflow resumes at the interrupted phase on next run.
- **Rollback support** — phases can roll back to earlier phases when concrete problems are discovered (e.g. requirements misunderstood, spec gaps).
- **Environment stripping** — `CLAUDECODE`, `CLAUDE_CODE_SSE_PORT`, and `CLAUDE_CODE_ENTRYPOINT` are removed from the child environment so Claude Code does not detect a nested session.
- **Signal forwarding** — `SIGWINCH` (terminal resize), `SIGINT`, and `SIGTERM` are forwarded to the child process. Window size changes are synced to the PTY.
- **Raw mode** — the user's terminal is placed into raw mode so keystrokes pass through unmodified, then restored on exit.

## Prerequisites

- **Go 1.24+** (uses the standard Go toolchain)
- **macOS or Linux** (PTY support via `github.com/creack/pty` — no Windows support)
- **Claude Code CLI** installed at `~/.local/bin/claude` or anywhere in `$PATH`
- **[Task](https://taskfile.dev/)** (optional, for build commands)

## Install

```bash
# Clone and build
git clone <repo-url> && cd claudemod
task build

# Binary is at bin/claudemod
./bin/claudemod

# Or install to ~/.local/bin
task install
```

If you don't have `task` installed, you can build directly:

```bash
go build -o bin/claudemod cmd/claudemod/main.go
```

## Usage

```bash
# Scaffold .claudemod/ and .claude/ in the current project (without launching Claude)
claudemod init

# Run a workflow
claudemod run <workflow-name>
```

### Commands

| Command                      | Description                                                                 |
| ---------------------------- | --------------------------------------------------------------------------- |
| `claudemod init`             | Scaffold `.claudemod/` and `.claude/` directories with workflow files       |
| `claudemod run <workflow>`   | Run a named workflow (e.g. `bootstrap`, `feature`)                          |

### Available workflows

| Workflow    | Phases                                                                                             | Purpose                                        |
| ----------- | -------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| `bootstrap` | bootstrap                                                                                          | Explore the codebase and generate initial specs |
| `feature`   | discuss-feature, spec-feature, scope-feature, tdd-red, tdd-green, code-review, synthesize-specs    | Full TDD feature development lifecycle          |

### Claude binary resolution

claudemod finds the `claude` binary in this order:

1. `~/.local/bin/claude` (the default Claude Code install location)
2. `claude` anywhere in `$PATH`

## The `.claudemod/` folder

Running `claudemod init` or `claudemod run` scaffolds a `.claudemod/` directory in your project:

```
.claudemod/
  WORKFLOW.md              Generated workflow definition (phase prompts and criteria)
  SESSION_STATE.json       Current phase and action (advance/restart/rollback/complete)
  FIX_PLAN.md              Task list with checkboxes for the current phase
  CHANGELOG.md             Dated entries summarizing completed work
  spec/                    Feature specs generated during workflows
    INDEX.md               Spec index
  refs/                    Reference templates
    SPEC.md                Template for individual specs
    SPEC_INDEX.md           Template for the spec index
```

It also ensures `.claude/settings.local.json` has read/write/edit permissions for the `.claudemod/` directory.

## Project structure

```
cmd/claudemod/main.go                     Entry point, subcommand dispatch
internal/
  app/
    app.go                                 App struct, workflow runner loop, session state I/O
    workflow.go                            Workflow and phase definitions (bootstrap, feature)
    claude_folder.go                       .claudemod/ and .claude/ scaffolding
    templates.go                           Go template rendering for prompts and workflow files
    files.go                               File/directory existence helpers
    prompt_system.go.tmpl                  System prompt template (phase instructions)
    workflow.go.tmpl                       Workflow definition template (WORKFLOW.md)
    refs/                                  Embedded reference templates (SPEC.md, SPEC_INDEX.md)
  launcher/
    launcher.go                            Spawns Claude Code sessions via the bridge
    claude.go                              Claude binary resolution
  claudecode/
    bridge/
      bridge.go                            PTY owner, raw mode, stdin pump, signal handler
      session.go                           Single PTY child process with output pump
      config.go                            Bridge and session config, env var stripping
      compat.go                            Platform compatibility
    terminal/
      rawmode.go                           Enter/restore raw terminal mode
      winsize.go                           Get/set window size (TIOCGWINSZ/TIOCSWINSZ)
    middleware/
      types.go                             Chunk, InputMiddleware, OutputMiddleware, Plugin
      pipeline.go                          Chain middleware execution
    config/config.go                       YAML config parsing
    plugin/
      registry.go                          Plugin name → factory registry
      loader.go                            Instantiate plugins from config
    ansi/parser.go                         ANSI escape sequence stripper
    plugins/
      logger/logger.go                     JSONL audit logging
      filter/filter.go                     Regex-based pattern redaction
      inject/inject.go                     Input text injection
    signals/signals.go                     SIGWINCH, SIGINT, SIGTERM forwarding
  utils/
    template.go                            Template helpers
    template_funcs.go                      Custom template functions
```

## Development

```bash
# Build
task build

# Run tests
task test

# Run tests with race detector
task test-race

# Run go vet + staticcheck
task lint

# Clean build artifacts
task clean
```

Or use `go` directly:

```bash
go build -o bin/claudemod cmd/claudemod/main.go
go test -cover ./...
go test -race ./...
go vet ./...
```

## Dependencies

| Module                      | Purpose                                       |
| --------------------------- | --------------------------------------------- |
| `github.com/creack/pty`     | PTY creation and management                   |
| `github.com/fsnotify/fsnotify` | File system watching for signal file detection |
| `github.com/segmentio/ksuid` | Unique session IDs                            |
| `github.com/google/uuid`    | UUID generation                               |
| `golang.org/x/term`         | Raw terminal mode                             |
| `golang.org/x/sys`          | ioctl for window size (TIOCGWINSZ/TIOCSWINSZ) |
| `gopkg.in/yaml.v3`          | YAML parsing                                  |
