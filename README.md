# claudemod

A developer-focused workflow builder that uses Claude Code to focus on incremental feature delivery for production-grade projects.

## How it works

ClaudeMod uses a Go PTY wrapper for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) that sits between your terminal and the Claude Code process. It intercepts all I/O for logging, redaction, and injection while preserving the full interactive TUI experience.

```
User Terminal <──stdin/stdout──> claudemod (PTY master) <──PTY slave──> Claude Code
                                       │
                                 Middleware Pipeline
                                 ├─ InputMiddleware[]   (user → claude)
                                 └─ OutputMiddleware[]   (claude → user)
```

claudemod spawns Claude Code on a pseudo-terminal (PTY) and holds the master end. Two goroutines pump data between your real terminal and the PTY — one for input (keystrokes) and one for output (TUI rendering). An optional middleware pipeline can observe or transform data in either direction.

Key behaviors:

- **Transparent by default** — with no config, claudemod is a pure passthrough. Claude Code looks and works identically.
- **Environment stripping** — `CLAUDECODE`, `CLAUDE_CODE_SSE_PORT`, and `CLAUDE_CODE_ENTRYPOINT` are removed from the child environment so Claude Code does not detect a nested session and refuse to start.
- **Signal forwarding** — `SIGWINCH` (terminal resize), `SIGINT`, and `SIGTERM` are forwarded to the child process. Window size changes are synced to the PTY.
- **Raw mode** — the user's terminal is placed into raw mode so keystrokes pass through unmodified to the PTY, then restored on exit.

## Prerequisites

- **Go 1.24+** (uses the standard Go toolchain)
- **macOS or Linux** (PTY support via `github.com/creack/pty` — no Windows support)
- **Claude Code CLI** installed at `~/.local/bin/claude` or anywhere in `$PATH`

## Install

```bash
# Clone and build
git clone <repo-url> && cd claudemod
make build

# Binary is at bin/claudemod
./bin/claudemod

# Or install to ~/.local/bin
make install
```

## Usage

```bash
# Run with no config — pure passthrough
claudemod

# Run with a config file
claudemod --config ~/.claudemod/config.yaml

# Pass arguments through to Claude Code (after --)
claudemod -- --model opus --continue

# Combine config and Claude args
claudemod --config ~/.claudemod/config.yaml -- --model opus
```

### Flags

| Flag              | Description                                                                                     |
| ----------------- | ----------------------------------------------------------------------------------------------- |
| `--config <path>` | Path to a YAML config file. Without this, claudemod runs as a pure passthrough with no plugins. |

Everything after `--` is forwarded directly to the Claude Code binary as arguments.

### Claude binary resolution

claudemod finds the `claude` binary in this order:

1. `claude_path` in the config file (if set)
2. `~/.local/bin/claude` (the default Claude Code install location)
3. `claude` anywhere in `$PATH`

## Configuration

Copy the example config and edit it:

```bash
mkdir -p ~/.claudemod
cp configs/claudemod.example.yaml ~/.claudemod/config.yaml
```

### Config format

```yaml
# Path to claude binary (optional, auto-detected if empty)
claude_path: ""

# Directory for log files
log_dir: "~/.claudemod/logs"

# Plugins — each has a name, enabled flag, and options map
plugins:
  - name: logger
    enabled: true
    options:
      log_input: true
      log_output: true

  - name: filter
    enabled: false
    options:
      patterns:
        - "sk-[a-zA-Z0-9]{20,}"
        - "AKIA[0-9A-Z]{16}"

  - name: inject
    enabled: false
    options:
      text: ""
```

Paths starting with `~/` are expanded to your home directory.

## Built-in plugins

### logger

Writes a JSONL audit log of all data flowing through the pipeline. ANSI escape sequences are stripped from logged data so logs contain clean text. Each session gets its own log file.

**Options:**

| Option       | Type   | Default             | Description                  |
| ------------ | ------ | ------------------- | ---------------------------- |
| `log_dir`    | string | `~/.claudemod/logs` | Directory for log files      |
| `log_input`  | bool   | `true`              | Log data from user to Claude |
| `log_output` | bool   | `true`              | Log data from Claude to user |

**Log format** (one JSON object per line):

```json
{
  "timestamp": "2026-02-17T20:15:03.123Z",
  "session_id": "abc-123",
  "direction": "output",
  "data": "Hello! How can I help?",
  "raw_len": 847
}
```

- `data` — ANSI-stripped text content
- `raw_len` — original byte count (including escape sequences)

The logger never modifies the data stream — it observes only.

### filter

Redacts sensitive patterns from both input and output using regex. Matched text is replaced with `[REDACTED]`.

**Options:**

| Option     | Type            | Default | Description              |
| ---------- | --------------- | ------- | ------------------------ |
| `patterns` | list of strings | `[]`    | Regex patterns to redact |

**Example patterns:**

```yaml
patterns:
  - "sk-[a-zA-Z0-9]{20,}" # OpenAI API keys
  - "AKIA[0-9A-Z]{16}" # AWS access key IDs
  - "ghp_[a-zA-Z0-9]{36}" # GitHub personal access tokens
```

When combined with the logger, the filter runs first in the pipeline so redacted values never reach the log files.

### inject

Prepends text to the first input chunk sent to Claude Code. Fires exactly once per session, then passes all subsequent input through unmodified.

**Options:**

| Option | Type   | Default | Description                        |
| ------ | ------ | ------- | ---------------------------------- |
| `text` | string | `""`    | Text to prepend to the first input |

## Plugin pipeline

Plugins execute in the order they appear in the config. Each plugin can implement `InputMiddleware`, `OutputMiddleware`, or both:

- **InputMiddleware** — transforms data flowing from user to Claude (`ProcessInput`)
- **OutputMiddleware** — transforms data flowing from Claude to user (`ProcessOutput`)

Data flows through the pipeline as immutable `Chunk` values. Middleware returns new chunks — it never mutates the original. If a middleware returns an empty chunk, the pipeline short-circuits and no further middleware runs for that chunk.

## Project structure

```
cmd/claudemod/main.go              Entry point, flag parsing, orchestration
internal/
  bridge/bridge.go                  PTY creation, child spawn, I/O pump goroutines
  terminal/rawmode.go               Enter/restore raw terminal mode
  terminal/winsize.go               Get/set window size (TIOCGWINSZ/TIOCSWINSZ)
  middleware/types.go               Chunk, InputMiddleware, OutputMiddleware, Plugin
  middleware/pipeline.go            Chain middleware execution
  signals/signals.go                SIGWINCH, SIGINT, SIGTERM forwarding
  config/config.go                  YAML config parsing
  plugin/registry.go                Plugin name → factory registry
  plugin/loader.go                  Instantiate plugins from config
  ansi/parser.go                    ANSI escape sequence stripper
  plugins/logger/logger.go          JSONL audit logging
  plugins/filter/filter.go          Regex-based pattern redaction
  plugins/inject/inject.go          Input text injection
```

## Development

```bash
# Build
make build

# Run tests
make test

# Run tests with race detector
make test-race

# Run go vet + staticcheck
make lint

# Clean build artifacts
make clean
```

### Writing a custom plugin

1. Create a new package under `internal/plugins/yourplugin/`.
2. Implement `middleware.Plugin` (the `Name()` method) plus `InputMiddleware` and/or `OutputMiddleware`.
3. Register it in an `init()` function:

```go
package yourplugin

import (
    "github.com/tbright/claudemod/internal/middleware"
    "github.com/tbright/claudemod/internal/plugin"
)

func init() {
    plugin.Register("yourplugin", newYourPlugin)
}

type YourPlugin struct{}

func newYourPlugin(opts map[string]any) (middleware.Plugin, error) {
    return &YourPlugin{}, nil
}

func (p *YourPlugin) Name() string { return "yourplugin" }

func (p *YourPlugin) ProcessOutput(chunk middleware.Chunk) middleware.Chunk {
    // Return chunk unmodified, or use chunk.WithData(newBytes) to transform
    return chunk
}
```

4. Add a blank import in `cmd/claudemod/main.go`:

```go
_ "github.com/tbright/claudemod/internal/plugins/yourplugin"
```

5. Add it to your config:

```yaml
plugins:
  - name: yourplugin
    enabled: true
    options: {}
```

If your plugin holds resources (open files, connections), implement `io.Closer` and it will be cleaned up on exit.

## Dependencies

| Module                   | Purpose                                       |
| ------------------------ | --------------------------------------------- |
| `github.com/creack/pty`  | PTY creation and management                   |
| `golang.org/x/term`      | Raw terminal mode                             |
| `golang.org/x/sys`       | ioctl for window size (TIOCGWINSZ/TIOCSWINSZ) |
| `gopkg.in/yaml.v3`       | YAML config parsing                           |
| `github.com/google/uuid` | Session IDs for log files                     |
