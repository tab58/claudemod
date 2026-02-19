package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const defaultClaudePath = "claude"

// resolveClaudePath finds the claude binary, preferring the well-known location.
func resolveClaudePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err == nil {
		localPath := filepath.Join(homeDir, ".local", "bin", "claude")
		if _, err := os.Stat(localPath); err == nil {
			return localPath, nil
		}
	}
	path, err := exec.LookPath(defaultClaudePath)
	if err != nil {
		return "", fmt.Errorf("claude binary not found in PATH or ~/.local/bin/claude")
	}
	return path, nil
}
