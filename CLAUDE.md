# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
make build                              # Build to bin/claudemod
make test                               # go test -cover ./...
make test-race                          # go test -race ./...
make lint                               # go vet + staticcheck (if installed)
make install                            # Build + copy to ~/.local/bin/claudemod
go test -run TestPTYRoundtrip ./internal/bridge/   # Run a single test
```

## Architecture

claudemod is a Go PTY wrapper that sits between the user's terminal and the Claude Code CLI. It spawns Claude Code on a pseudo-terminal slave, holds the master end, and pumps I/O through an optional middleware pipeline in two goroutines (input and output).

### Data flow

```
stdin → [InputMiddleware pipeline] → PTY master → Claude Code (PTY slave)
Claude Code (PTY slave) → PTY master → [OutputMiddleware pipeline] → stdout
```

### Core subsystems

**Bridge** (`internal/bridge/`) — Owns the PTY lifecycle. Creates the child process with env vars stripped (`CLAUDECODE`, `CLAUDE_CODE_SSE_PORT`, `CLAUDE_CODE_ENTRYPOINT`) so Claude Code doesn't detect nesting. Puts the terminal in raw mode, syncs window size, and runs two pump goroutines. Accepts `ProcessInput`/`ProcessOutput` callbacks from the middleware pipeline.

**Middleware** (`internal/middleware/`) — Defines the immutable `Chunk` type and `Pipeline`. All data flows as `Chunk` values where `Data()` returns a copy and `WithData()` returns a new chunk. Plugins implement `InputMiddleware`, `OutputMiddleware`, or both. The pipeline short-circuits if any middleware returns an empty chunk.

**Plugin system** (`internal/plugin/`) — Registry + Factory pattern. Plugins self-register via `init()` in their packages. `main.go` triggers registration through blank imports. `LoadAll()` instantiates enabled plugins from config in order. Plugins implementing `io.Closer` are cleaned up on exit via `middleware.CloseAll()`.

**Signals** (`internal/signals/`) — Forwards SIGWINCH (with PTY resize), SIGINT, and SIGTERM to the child process.

### Adding a new plugin

1. Create `internal/plugins/yourplugin/yourplugin.go`
2. Call `plugin.Register("yourplugin", factory)` in `init()`
3. Implement `middleware.Plugin` (`Name()`) plus `InputMiddleware` and/or `OutputMiddleware`
4. Add blank import in `cmd/claudemod/main.go`: `_ "github.com/tbright/claudemod/internal/plugins/yourplugin"`
5. Add entry to config YAML under `plugins:`

### Built-in plugins

- **logger** — Observe-only JSONL audit logging with ANSI stripping. Thread-safe via mutex. Implements `io.Closer`.
- **filter** — Regex-based redaction replacing matches with `[REDACTED]` on both input and output.
- **inject** — Prepends text to the first input chunk using `sync.Once`, then passes through.

## Key Constraints

- **macOS/Linux only** — PTY support, unix ioctls, SIGWINCH are not available on Windows.
- **Immutability** — Never mutate a `Chunk`. Use `WithData()` to create transformed copies. `Data()` always returns a defensive copy.
- **Environment stripping** — The child process must not receive `CLAUDECODE=1` or related vars, otherwise Claude Code refuses to start.
- **Module path**: `github.com/tbright/claudemod`, Go 1.24+.
