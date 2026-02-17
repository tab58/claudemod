package terminal

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// WinSize represents terminal window dimensions.
type WinSize struct {
	Rows uint16
	Cols uint16
}

// GetWinSize reads the current window size from the given file descriptor.
func GetWinSize(f *os.File) (WinSize, error) {
	ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return WinSize{}, fmt.Errorf("get window size: %w", err)
	}
	return WinSize{Rows: ws.Row, Cols: ws.Col}, nil
}

// SetWinSize sets the window size on the given file descriptor.
func SetWinSize(f *os.File, size WinSize) error {
	ws := &unix.Winsize{Row: size.Rows, Col: size.Cols}
	err := unix.IoctlSetWinsize(int(f.Fd()), unix.TIOCSWINSZ, ws)
	if err != nil {
		return fmt.Errorf("set window size: %w", err)
	}
	return nil
}
