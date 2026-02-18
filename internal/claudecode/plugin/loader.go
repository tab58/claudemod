package plugin

import (
	"fmt"

	"github.com/tab58/claudemod/internal/claudecode/config"
	"github.com/tab58/claudemod/internal/claudecode/middleware"
)

// LoadAll instantiates all enabled plugins from the config, using the global registry.
func LoadAll(cfg config.Config) ([]middleware.Plugin, error) {
	var plugins []middleware.Plugin

	for _, pc := range cfg.Plugins {
		if !pc.Enabled {
			continue
		}

		factory, err := Get(pc.Name)
		if err != nil {
			return nil, fmt.Errorf("load plugin %q: %w", pc.Name, err)
		}

		p, err := factory(pc.Options)
		if err != nil {
			return nil, fmt.Errorf("initialize plugin %q: %w", pc.Name, err)
		}

		plugins = append(plugins, p)
	}

	return plugins, nil
}
