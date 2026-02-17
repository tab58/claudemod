package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ClaudePath != "" {
		t.Errorf("expected empty ClaudePath, got %q", cfg.ClaudePath)
	}
	if cfg.LogDir != "~/.claudemod/logs" {
		t.Errorf("expected default LogDir, got %q", cfg.LogDir)
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	content := `
claude_path: /usr/local/bin/claude
log_dir: /tmp/logs
plugins:
  - name: logger
    enabled: true
    options:
      log_input: true
  - name: filter
    enabled: false
`
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.ClaudePath != "/usr/local/bin/claude" {
		t.Errorf("expected ClaudePath '/usr/local/bin/claude', got %q", cfg.ClaudePath)
	}
	if cfg.LogDir != "/tmp/logs" {
		t.Errorf("expected LogDir '/tmp/logs', got %q", cfg.LogDir)
	}
	if len(cfg.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(cfg.Plugins))
	}
	if cfg.Plugins[0].Name != "logger" || !cfg.Plugins[0].Enabled {
		t.Error("first plugin should be logger, enabled")
	}
	if cfg.Plugins[1].Name != "filter" || cfg.Plugins[1].Enabled {
		t.Error("second plugin should be filter, disabled")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"tilde path", "~/foo/bar", home + "/foo/bar"},
		{"absolute path", "/usr/local", "/usr/local"},
		{"relative path", "foo/bar", "foo/bar"},
		{"empty", "", ""},
		{"tilde only", "~", "~"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandHome(tt.input)
			if got != tt.expected {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
