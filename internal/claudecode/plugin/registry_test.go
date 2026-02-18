package plugin

import (
	"testing"

	"github.com/tab58/claudemod/internal/claudecode/middleware"
)

type stubPlugin struct{ name string }

func (s stubPlugin) Name() string { return s.name }

func resetRegistry() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Factory)
}

func TestRegisterAndGet(t *testing.T) {
	resetRegistry()
	Register("test", func(opts map[string]any) (middleware.Plugin, error) {
		return stubPlugin{name: "test"}, nil
	})

	factory, err := Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	p, err := factory(nil)
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if p.Name() != "test" {
		t.Errorf("expected name 'test', got %q", p.Name())
	}
}

func TestGet_NotFound(t *testing.T) {
	resetRegistry()
	_, err := Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown plugin")
	}
}

func TestRegister_Duplicate_Panics(t *testing.T) {
	resetRegistry()
	factory := func(opts map[string]any) (middleware.Plugin, error) {
		return stubPlugin{}, nil
	}
	Register("dup", factory)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate registration")
		}
	}()
	Register("dup", factory)
}

func TestNames(t *testing.T) {
	resetRegistry()
	Register("alpha", func(opts map[string]any) (middleware.Plugin, error) {
		return stubPlugin{}, nil
	})
	Register("beta", func(opts map[string]any) (middleware.Plugin, error) {
		return stubPlugin{}, nil
	})

	names := Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}

	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["alpha"] || !found["beta"] {
		t.Errorf("expected alpha and beta, got %v", names)
	}
}
