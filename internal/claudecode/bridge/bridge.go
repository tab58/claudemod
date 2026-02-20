package bridge

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/tab58/claudemod/internal/claudecode/terminal"
)

// Bridge owns the user's terminal (raw mode, stdin reader, signal handler)
// and manages one or more Sessions. At most one Session is active at a time
// and receives stdin / writes stdout.
type Bridge struct {
	stdin      *os.File
	stdout     io.Writer
	rawState   terminal.RawModeState
	mu         sync.Mutex
	activePtr  atomic.Pointer[Session]
	sessions   []*Session
	signalDone chan struct{}  // closed by Close to stop signalHandler
	signalWg   sync.WaitGroup // tracks signalHandler goroutine
	stdinDone  chan struct{}  // closed when stdinPump exits
	closed     bool
}

// New creates a Bridge that owns the terminal. It enters raw mode and
// starts the stdin pump and signal handler goroutines.
func New(cfg BridgeConfig) (*Bridge, error) {
	stdin := cfg.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := cfg.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	rawState, err := terminal.EnableRawMode(stdin)
	if err != nil {
		return nil, fmt.Errorf("enable raw mode: %w", err)
	}

	b := &Bridge{
		stdin:      stdin,
		stdout:     stdout,
		rawState:   rawState,
		signalDone: make(chan struct{}),
		stdinDone:  make(chan struct{}),
	}

	go b.stdinPump()

	b.signalWg.Add(1)
	go b.signalHandler()

	return b, nil
}

// Spawn creates a new Session with the given command. The session is not
// activated — call Activate to connect it to stdin/stdout.
func (b *Bridge) Spawn(path string, args []string, cfg Config) (*Session, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, fmt.Errorf("bridge closed")
	}

	s, err := newSession(b, path, args, cfg)
	if err != nil {
		return nil, err
	}
	b.sessions = append(b.sessions, s)
	return s, nil
}

// Activate makes s the active session. The previous session (if any) is
// deactivated — its output pump parks. Pass nil to deactivate all sessions.
func (b *Bridge) Activate(s *Session) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if prev := b.activePtr.Load(); prev != nil {
		prev.setActive(false)
	}

	if s != nil {
		if ws, err := terminal.GetWinSize(b.stdin); err == nil {
			_ = terminal.SetWinSize(s.pty, ws)
		}
		s.setActive(true)
	}
	b.activePtr.Store(s)

	return nil
}

// Active returns the currently active Session, or nil.
func (b *Bridge) Active() *Session {
	return b.activePtr.Load()
}

// Close shuts down the Bridge: stops signal forwarding, closes all sessions,
// waits for the signal handler goroutine, and restores the terminal.
// The stdinPump goroutine terminates when the stdin file descriptor is
// closed by the caller (or on process exit).
func (b *Bridge) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	b.activePtr.Store(nil)
	close(b.signalDone)

	for _, s := range b.sessions {
		s.Close()
	}

	b.signalWg.Wait()

	return b.rawState.Restore()
}

// stdinPump is a single goroutine that reads stdin and writes to the active
// session's PTY. Uses atomic load (no lock) for the hot path.
// Terminates when stdin returns an error (EOF / closed fd) or on PTY write error.
func (b *Bridge) stdinPump() {
	defer close(b.stdinDone)
	buf := make([]byte, 32*1024)
	for {
		n, err := b.stdin.Read(buf)
		if n > 0 {
			s := b.activePtr.Load()
			if s != nil {
				data := make([]byte, n)
				copy(data, buf[:n])
				if s.processInput != nil {
					data = s.processInput(data)
				}
				if len(data) > 0 {
					if _, werr := s.pty.Write(data); werr != nil {
						return
					}
				}
			}
		}
		if err != nil {
			return
		}
	}
}

// signalHandler routes SIGWINCH, SIGINT, SIGTERM to the active session.
func (b *Bridge) signalHandler() {
	defer b.signalWg.Done()

	winchCh := make(chan os.Signal, 1)
	signal.Notify(winchCh, syscall.SIGWINCH)

	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, syscall.SIGINT, syscall.SIGTERM)

	defer signal.Stop(winchCh)
	defer signal.Stop(termCh)

	for {
		select {
		case <-b.signalDone:
			return
		case <-winchCh:
			s := b.activePtr.Load()
			if s == nil || s.cmd.Process == nil {
				continue
			}
			if ws, err := terminal.GetWinSize(b.stdin); err == nil {
				_ = terminal.SetWinSize(s.pty, ws)
			}
			_ = s.cmd.Process.Signal(syscall.SIGWINCH)
		case sig := <-termCh:
			s := b.activePtr.Load()
			if s == nil || s.cmd.Process == nil {
				continue
			}
			if ss, ok := sig.(syscall.Signal); ok {
				_ = s.cmd.Process.Signal(ss)
			}
		}
	}
}
