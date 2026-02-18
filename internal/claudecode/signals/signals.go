package signals

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/tab58/claudemod/internal/claudecode/terminal"
)

// Deprecated: ForwardTo is superseded by Bridge.signalHandler which routes
// signals to the active session. Kept for external reference only.
//
// ForwardTo starts goroutines that forward relevant signals to the child process
// and handle SIGWINCH by resizing the PTY.
// Returns a cancel function that stops signal forwarding.
func ForwardTo(child *os.Process, stdinFile *os.File, ptyMaster *os.File) func() {
	winchCh := make(chan os.Signal, 1)
	signal.Notify(winchCh, syscall.SIGWINCH)

	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-winchCh:
				ws, err := terminal.GetWinSize(stdinFile)
				if err != nil {
					continue
				}
				_ = terminal.SetWinSize(ptyMaster, ws)
				_ = child.Signal(syscall.SIGWINCH)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-done:
				return
			case sig := <-termCh:
				_ = child.Signal(sig.(syscall.Signal))
			}
		}
	}()

	return func() {
		signal.Stop(winchCh)
		signal.Stop(termCh)
		close(done)
	}
}
