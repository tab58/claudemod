package inject

import (
	"testing"

	"github.com/tab58/claudemod/internal/claudecode/middleware"
	"github.com/tab58/claudemod/internal/claudecode/plugin"
)

func TestInject_PrependOnFirstChunk(t *testing.T) {
	factory, err := plugin.Get("inject")
	if err != nil {
		t.Fatalf("inject not registered: %v", err)
	}

	p, err := factory(map[string]any{"text": "PREFIX:"})
	if err != nil {
		t.Fatalf("create inject: %v", err)
	}

	im := p.(middleware.InputMiddleware)

	// First chunk should have prefix
	chunk1 := middleware.NewChunk([]byte("hello"), middleware.Input)
	result1 := im.ProcessInput(chunk1)
	if string(result1.Data()) != "PREFIX:hello" {
		t.Errorf("expected 'PREFIX:hello', got %q", result1.Data())
	}

	// Second chunk should pass through
	chunk2 := middleware.NewChunk([]byte("world"), middleware.Input)
	result2 := im.ProcessInput(chunk2)
	if string(result2.Data()) != "world" {
		t.Errorf("expected 'world', got %q", result2.Data())
	}
}

func TestInject_EmptyText_Passthrough(t *testing.T) {
	factory, _ := plugin.Get("inject")
	p, err := factory(map[string]any{"text": ""})
	if err != nil {
		t.Fatal(err)
	}

	im := p.(middleware.InputMiddleware)
	chunk := middleware.NewChunk([]byte("hello"), middleware.Input)
	result := im.ProcessInput(chunk)

	if string(result.Data()) != "hello" {
		t.Errorf("expected passthrough, got %q", result.Data())
	}
}

func TestInject_NoTextOption_Passthrough(t *testing.T) {
	factory, _ := plugin.Get("inject")
	p, err := factory(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	im := p.(middleware.InputMiddleware)
	chunk := middleware.NewChunk([]byte("hello"), middleware.Input)
	result := im.ProcessInput(chunk)

	if string(result.Data()) != "hello" {
		t.Errorf("expected passthrough, got %q", result.Data())
	}
}
