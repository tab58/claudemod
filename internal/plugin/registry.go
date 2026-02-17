package plugin

import (
	"fmt"
	"sync"

	"github.com/tbright/claudemod/internal/middleware"
)

// Factory creates a Plugin instance from the given options map.
type Factory func(options map[string]any) (middleware.Plugin, error)

var (
	mu       sync.RWMutex
	registry = make(map[string]Factory)
)

// Register adds a named plugin factory to the global registry.
// Typically called from plugin init() functions.
func Register(name string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("plugin %q already registered", name))
	}
	registry[name] = factory
}

// Get returns the factory for the named plugin, or an error if not found.
func Get(name string) (Factory, error) {
	mu.RLock()
	defer mu.RUnlock()
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown plugin: %q", name)
	}
	return f, nil
}

// Names returns all registered plugin names.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
