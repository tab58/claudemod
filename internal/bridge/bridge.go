package bridge

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/creack/pty"

	"github.com/tbright/claudemod/internal/signals"
	"github.com/tbright/claudemod/internal/terminal"
)

// strippedEnvVars are removed from the child environment to prevent
// Claude Code from detecting a nested session.
var strippedEnvVars = []string{
	"CLAUDECODE",
	"CLAUDE_CODE_SSE_PORT",
	"CLAUDE_CODE_ENTRYPOINT",
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

// Run spawns the given command on a PTY and bridges I/O between
// the user's terminal and the child process. It returns the child's exit code.
func Run(claudePath string, args []string, cfg Config) (int, error) {
	cmd := exec.Command(claudePath, args...)
	cmd.Env = cleanEnv(os.Environ())

	// Start child on a new PTY.
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return 1, fmt.Errorf("start pty: %w", err)
	}
	defer ptmx.Close()

	// Sync initial window size from the user's terminal to the PTY.
	if ws, err := terminal.GetWinSize(os.Stdin); err == nil {
		_ = terminal.SetWinSize(ptmx, ws)
	}

	// Put user's terminal into raw mode so keystrokes pass through.
	rawState, err := terminal.EnableRawMode(os.Stdin)
	if err != nil {
		return 1, fmt.Errorf("enable raw mode: %w", err)
	}
	defer rawState.Restore()

	// Forward signals (SIGWINCH, SIGINT, SIGTERM) to the child.
	cancelSignals := signals.ForwardTo(cmd.Process, os.Stdin, ptmx)
	defer cancelSignals()

	// Pump I/O between user terminal and child PTY.
	var wg sync.WaitGroup
	wg.Add(2)

	// stdin -> PTY (input)
	go func() {
		defer wg.Done()
		pump(os.Stdin, ptmx, cfg.ProcessInput)
	}()

	// PTY -> stdout (output)
	go func() {
		defer wg.Done()
		pump(ptmx, os.Stdout, cfg.ProcessOutput)
	}()

	// Wait for child to exit.
	err = cmd.Wait()
	wg.Wait()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("wait for child: %w", err)
	}
	return 0, nil
}

// pump copies data from src to dst, optionally passing through a transform.
func pump(src io.Reader, dst io.Writer, transform func([]byte) []byte) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if transform != nil {
				data := make([]byte, n)
				copy(data, buf[:n])
				data = transform(data)
				if len(data) > 0 {
					if _, werr := dst.Write(data); werr != nil {
						return
					}
				}
			} else {
				if _, werr := dst.Write(buf[:n]); werr != nil {
					return
				}
			}
		}
		if err != nil {
			return
		}
	}
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
