package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration structure.
type Config struct {
	ClaudePath string         `yaml:"claude_path"`
	LogDir     string         `yaml:"log_dir"`
	Plugins    []PluginConfig `yaml:"plugins"`
}

// PluginConfig describes a single plugin instance with its settings.
type PluginConfig struct {
	Name    string         `yaml:"name"`
	Enabled bool           `yaml:"enabled"`
	Options map[string]any `yaml:"options"`
}

// DefaultConfig returns a Config with sensible defaults and no plugins enabled.
func DefaultConfig() Config {
	return Config{
		ClaudePath: "",
		LogDir:     "~/.claudemod/logs",
	}
}

// Load reads and parses a YAML config file, returning the populated Config.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}

	return cfg, nil
}

// ExpandHome replaces a leading "~/" with the user's home directory.
func ExpandHome(path string) string {
	if len(path) < 2 || path[:2] != "~/" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return home + path[1:]
}
