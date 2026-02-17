package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/tbright/claudemod/internal/bridge"
	"github.com/tbright/claudemod/internal/config"
	"github.com/tbright/claudemod/internal/middleware"
	"github.com/tbright/claudemod/internal/plugin"

	// Register built-in plugins via init().
	_ "github.com/tbright/claudemod/internal/plugins/filter"
	_ "github.com/tbright/claudemod/internal/plugins/inject"
	_ "github.com/tbright/claudemod/internal/plugins/logger"
)

const defaultClaudePath = "claude"

func main() {
	configPath := flag.String("config", "", "path to config YAML file")
	flag.Parse()

	cfg := config.DefaultConfig()
	if *configPath != "" {
		var err error
		cfg, err = config.Load(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
			os.Exit(1)
		}
	}

	claudePath := cfg.ClaudePath
	if claudePath == "" {
		var err error
		claudePath, err = resolveClaudePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
			os.Exit(1)
		}
	}

	// Load plugins from config.
	plugins, err := plugin.LoadAll(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
		os.Exit(1)
	}
	defer middleware.CloseAll(plugins)
	pipeline := middleware.NewPipeline(plugins)

	// Everything after "--" or remaining flag.Args() goes to claude.
	claudeArgs := extractClaudeArgs(flag.Args())

	bridgeCfg := bridge.Config{
		ProcessInput:  pipeline.InputFunc(),
		ProcessOutput: pipeline.OutputFunc(),
	}

	exitCode, err := bridge.Run(claudePath, claudeArgs, bridgeCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
		os.Exit(1)
	}
	os.Exit(exitCode)
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
