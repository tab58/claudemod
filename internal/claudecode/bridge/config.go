package bridge

import (
	"io"
	"os"
	"strings"
)

// strippedEnvVars are removed from the child environment to prevent
// Claude Code from detecting a nested session.
var strippedEnvVars = []string{
	"CLAUDECODE",
	"CLAUDE_CODE_SSE_PORT",
	"CLAUDE_CODE_ENTRYPOINT",
}

// BridgeConfig configures the terminal-owning Bridge.
type BridgeConfig struct {
	// Stdin is the terminal file descriptor for raw mode and winsize.
	// If nil, defaults to os.Stdin.
	Stdin *os.File

	// Stdout receives output from the active session.
	// If nil, defaults to os.Stdout.
	// Using io.Writer (not *os.File) because stdout never needs ioctls.
	Stdout io.Writer
}

// Config holds optional callbacks for I/O transformation.
// When nil, data passes through unmodified.
type Config struct {
	// ProcessInput transforms data flowing from user stdin to the child PTY.
	// Receives raw bytes, returns bytes to write. Return nil to drop.
	ProcessInput func([]byte) []byte

	// ProcessOutput transforms data flowing from the child PTY to user stdout.
	// Receives raw bytes, returns bytes to write. Return nil to drop.
	ProcessOutput func([]byte) []byte
}

// cleanEnv returns a copy of the environment with Claude nesting vars removed.
func cleanEnv(env []string) []string {
	result := make([]string, 0, len(env))
	for _, e := range env {
		stripped := false
		for _, prefix := range strippedEnvVars {
			if strings.HasPrefix(e, prefix+"=") {
				stripped = true
				break
			}
		}
		if !stripped {
			result = append(result, e)
		}
	}
	return result
}
