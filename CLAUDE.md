# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
task build                              # Build to bin/claudemod
task test                               # go test -cover ./...
task test-race                          # go test -race ./...
task lint                               # go vet + staticcheck (if installed)
task install                            # Build + copy to ~/.local/bin/claudemod
go test -run TestName ./internal/path/  # Run a single test
```

If `task` is not installed, use `go` directly:

```bash
go build -o bin/claudemod cmd/claudemod/main.go
go test -cover ./...
```

## Architecture

claudemod is a workflow orchestrator for Claude Code that drives incremental feature delivery through structured, multi-phase development workflows. It spawns sequential Claude Code sessions, each with a phase-specific system prompt, on a PTY bridge that sits between the user's terminal and the Claude Code process.

### Two main subsystems

**Workflow engine** (`internal/app/`) — Defines workflows as ordered sequences of phases. Each phase spawns an interactive Claude Code session with a generated system prompt. Phase transitions are driven by `SESSION_STATE.json` — Claude writes an action (`advance`, `rollback`, `restart`, `complete`) and the workflow loop reads it after each phase to decide what happens next.

**PTY bridge** (`internal/claudecode/bridge/`) — Owns the user's terminal in raw mode, manages PTY child processes. At most one Session is active at a time. The Bridge runs a `stdinPump` goroutine that routes user input to the active session's PTY, and each Session runs an `outputPump` goroutine that reads from the PTY and writes to stdout. Session activation is gated by a channel token — data produced while inactive is discarded.

### Data flow

```
stdin → Bridge.stdinPump → [processInput callback] → PTY master → Claude Code (PTY slave)
Claude Code → PTY master → Session.outputPump → [processOutput callback] → stdout
```

### Phase transition flow

```
App.RunWorkflow → read SESSION_STATE.json (starting phase)
  → for each phase:
      → generate system prompt from template
      → Launcher.SpawnInteractiveSession
          → build claude CLI args (--append-system-prompt, --add-dir, -- "Begin.")
          → Bridge.Spawn + Bridge.Activate
          → watch for signal_<ksuid> file via fsnotify
          → block until: signal file created | child exits | ctx cancelled
      → read SESSION_STATE.json → switch on action field
```

The signal file is the cooperative termination protocol: Claude writes `signal_<ksuid>` to the working directory to indicate phase completion. The launcher detects this via `fsnotify`, suspends the child (SIGSTOP), then closes it.

### Plugin system

Registry + Factory pattern in `internal/claudecode/plugin/`. Plugins self-register via `init()` and implement `InputMiddleware`, `OutputMiddleware`, or both from `internal/claudecode/middleware/`. The middleware pipeline chains plugins in order and short-circuits if any returns an empty chunk.

**Note**: The plugin infrastructure is built but not yet wired to the launcher — `SpawnInteractiveSession` passes empty `bridge.Config{}`. To connect plugins, load them via `plugin.LoadAll()`, build a `middleware.Pipeline`, and pass `Pipeline.InputFunc()`/`OutputFunc()` as the `bridge.Config` callbacks.

Built-in plugins: **logger** (JSONL audit log with ANSI stripping), **filter** (regex redaction), **inject** (prepend text to first input chunk).

### Adding a new plugin

1. Create `internal/claudecode/plugins/yourplugin/yourplugin.go`
2. Call `plugin.Register("yourplugin", factory)` in `init()`
3. Implement `middleware.Plugin` (`Name()`) plus `InputMiddleware` and/or `OutputMiddleware`
4. If the plugin holds resources, implement `io.Closer`

## Key Constraints

- **macOS/Linux only** — PTY support, unix ioctls, SIGWINCH are not available on Windows.
- **Immutability** — Never mutate a `Chunk`. Use `WithData()` to create transformed copies. `Data()` always returns a defensive copy.
- **Environment stripping** — `cleanEnv()` in `bridge/config.go` strips `CLAUDECODE`, `CLAUDE_CODE_SSE_PORT`, and `CLAUDE_CODE_ENTRYPOINT` from the child environment. If these reach Claude Code, it detects nesting and refuses to start.
- **Module path**: `github.com/tab58/claudemod`, Go 1.24+.
- **Claude binary resolution**: checks `~/.local/bin/claude` first, then `$PATH`.
- **`signals.ForwardTo`** in `internal/claudecode/signals/` is deprecated — signal handling lives in `Bridge.signalHandler`.
