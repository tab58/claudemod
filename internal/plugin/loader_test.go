package plugin

import (
	"fmt"
	"testing"

	"github.com/tbright/claudemod/internal/config"
	"github.com/tbright/claudemod/internal/middleware"
)

func TestLoadAll_EnabledPlugins(t *testing.T) {
	resetRegistry()
	Register("testplugin", func(opts map[string]any) (middleware.Plugin, error) {
		return stubPlugin{name: "testplugin"}, nil
	})

	cfg := config.Config{
		Plugins: []config.PluginConfig{
			{Name: "testplugin", Enabled: true},
		},
	}

	plugins, err := LoadAll(cfg)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name() != "testplugin" {
		t.Errorf("expected 'testplugin', got %q", plugins[0].Name())
	}
}

func TestLoadAll_SkipsDisabled(t *testing.T) {
	resetRegistry()
	Register("skip", func(opts map[string]any) (middleware.Plugin, error) {
		return stubPlugin{name: "skip"}, nil
	})

	cfg := config.Config{
		Plugins: []config.PluginConfig{
			{Name: "skip", Enabled: false},
		},
	}

	plugins, err := LoadAll(cfg)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestLoadAll_UnknownPlugin(t *testing.T) {
	resetRegistry()
	cfg := config.Config{
		Plugins: []config.PluginConfig{
			{Name: "nonexistent", Enabled: true},
		},
	}

	_, err := LoadAll(cfg)
	if err == nil {
		t.Error("expected error for unknown plugin")
	}
}

func TestLoadAll_FactoryError(t *testing.T) {
	resetRegistry()
	Register("broken", func(opts map[string]any) (middleware.Plugin, error) {
		return nil, fmt.Errorf("initialization failed")
	})

	cfg := config.Config{
		Plugins: []config.PluginConfig{
			{Name: "broken", Enabled: true},
		},
	}

	_, err := LoadAll(cfg)
	if err == nil {
		t.Error("expected error for broken factory")
	}
}

func TestLoadAll_EmptyConfig(t *testing.T) {
	resetRegistry()
	cfg := config.Config{}

	plugins, err := LoadAll(cfg)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if plugins != nil {
		t.Errorf("expected nil plugins, got %v", plugins)
	}
}
