package terminal

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// RawModeState holds the saved terminal state for restoration.
type RawModeState struct {
	fd    int
	state *term.State
}

// EnableRawMode puts the given file descriptor into raw mode and returns
// the saved state needed to restore it later.
func EnableRawMode(f *os.File) (RawModeState, error) {
	fd := int(f.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return RawModeState{}, fmt.Errorf("enable raw mode: %w", err)
	}
	return RawModeState{fd: fd, state: state}, nil
}

// Restore reverts the terminal to its original state.
func (s RawModeState) Restore() error {
	if s.state == nil {
		return nil
	}
	return term.Restore(s.fd, s.state)
}
