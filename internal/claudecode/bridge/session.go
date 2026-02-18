package bridge

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/creack/pty"

	"github.com/tab58/claudemod/internal/claudecode/terminal"
)

// Session owns a single PTY child process. Its output pump is gated:
// when inactive the pump parks, when active it reads and writes to the
// Bridge's stdout.
type Session struct {
	bridge        *Bridge
	cmd           *exec.Cmd
	pty           *os.File
	processInput  func([]byte) []byte
	processOutput func([]byte) []byte
	isActive      atomic.Bool
	// outputGate has capacity 1. A token is sent on activation and drained
	// on deactivation. The pump blocks on receive when inactive. Closing the
	// channel signals the pump to exit.
	outputGate chan struct{}
	done       chan struct{}
	exitMu     sync.Mutex
	exitSet    bool
	exitCode   int
	exitErr    error
	closeOnce  sync.Once
}

// newSession creates a child process on a new PTY, syncs the initial
// window size, starts the output pump, and starts a wait goroutine.
func newSession(b *Bridge, path string, args []string, cfg Config) (*Session, error) {
	cmd := exec.Command(path, args...)
	cmd.Env = cleanEnv(os.Environ())

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("start pty: %w", err)
	}

	if ws, wsErr := terminal.GetWinSize(b.stdin); wsErr == nil {
		_ = terminal.SetWinSize(ptmx, ws)
	}

	s := &Session{
		bridge:        b,
		cmd:           cmd,
		pty:           ptmx,
		processInput:  cfg.ProcessInput,
		processOutput: cfg.ProcessOutput,
		outputGate:    make(chan struct{}, 1),
		done:          make(chan struct{}),
	}

	go s.outputPump()
	go s.waitChild()

	return s, nil
}

// outputPump reads from the PTY master and writes to the Bridge's stdout.
// When inactive, the pump parks on the outputGate channel. Data read while
// the session is inactive (between Read returning and the isActive check) is
// intentionally discarded — at most one 32KB chunk, which is benign for
// terminal I/O in a multiplexer.
func (s *Session) outputPump() {
	buf := make([]byte, 32*1024)
	for {
		// Park while inactive.
		if !s.isActive.Load() {
			if _, ok := <-s.outputGate; !ok {
				return // session closed
			}
		}

		n, err := s.pty.Read(buf)
		if n > 0 && s.isActive.Load() {
			data := make([]byte, n)
			copy(data, buf[:n])
			if s.processOutput != nil {
				data = s.processOutput(data)
			}
			if len(data) > 0 {
				if _, werr := s.bridge.stdout.Write(data); werr != nil {
					return
				}
			}
		}
		if err != nil {
			return
		}
	}
}

// waitChild blocks until the child exits and records the result.
func (s *Session) waitChild() {
	err := s.cmd.Wait()

	s.exitMu.Lock()
	s.exitSet = true
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			s.exitCode = exitErr.ExitCode()
		} else {
			s.exitCode = 1
			s.exitErr = fmt.Errorf("wait for child: %w", err)
		}
	}
	s.exitMu.Unlock()

	close(s.done)
}

// setActive controls the output pump gate. When activated, a token is sent
// to wake the parked pump. When deactivated, any stale token is drained to
// prevent spurious unparking on the next activation cycle.
func (s *Session) setActive(active bool) {
	s.isActive.Store(active)
	if active {
		select {
		case s.outputGate <- struct{}{}:
		default:
		}
	} else {
		// Drain stale token to prevent spurious unpark.
		select {
		case <-s.outputGate:
		default:
		}
	}
}

// Suspend sends SIGSTOP to the child process. Suspend and Resume are
// orthogonal to activation — the caller is responsible for coordinating
// Activate, Suspend, and Resume in the correct order.
func (s *Session) Suspend() error {
	if s.cmd.Process == nil {
		return fmt.Errorf("session: no process")
	}
	return s.cmd.Process.Signal(syscall.SIGSTOP)
}

// Resume sends SIGCONT to the child process.
func (s *Session) Resume() error {
	if s.cmd.Process == nil {
		return fmt.Errorf("session: no process")
	}
	return s.cmd.Process.Signal(syscall.SIGCONT)
}

// Wait blocks until the child exits and returns its exit code.
// A non-zero exit code is not reported as an error — only unexpected
// failures (e.g. process not started) set the error return.
func (s *Session) Wait() (int, error) {
	<-s.done
	s.exitMu.Lock()
	defer s.exitMu.Unlock()
	return s.exitCode, s.exitErr
}

// Done returns a channel that closes when the child exits.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

// close deactivates the session, then shuts down the output gate and PTY.
func (s *Session) close() {
	s.closeOnce.Do(func() {
		s.setActive(false)
		close(s.outputGate)
		s.pty.Close()
	})
}
