package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/tab58/claudemod/internal/launcher"
)

const defaultClaudePath = "claude"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
		os.Exit(1)
	}
	lnch, err := launcher.New(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
		os.Exit(1)
	}
	defer lnch.Close()

	fmt.Println("spawning self-terminating session: say the word 'exit' to exit.")

	if err := lnch.SpawnInteractiveSession(ctx, launcher.SessionParams{
		Prompt:       "Start by telling the user 'Hello!'.",
		ExitCriteria: "Say the phrase 'close this session'.",
	}); err != nil {
		if ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "claudemod: session error: %v\n", err)
			os.Exit(1)
		}
	}

	// claudeCode, err = b.Spawn("claude", []string{}, bridge.Config{})
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
	// 	os.Exit(1)
	// }
	// b.Activate(claudeCode)
	// exitCode, err := claudeCode.Wait()
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
	// 	os.Exit(1)
	// }
	// os.Exit(exitCode)
}

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

// extractClaudeArgs returns everything after "--" or all args if no separator found.
func extractClaudeArgs(args []string) []string {
	for i, arg := range args {
		if arg == "--" {
			return args[i+1:]
		}
	}
	return args
}
