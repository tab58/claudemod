package filter

import (
	"testing"

	"github.com/tab58/claudemod/internal/claudecode/middleware"
	"github.com/tab58/claudemod/internal/claudecode/plugin"
)

func TestFilter_RedactsPatterns(t *testing.T) {
	factory, err := plugin.Get("filter")
	if err != nil {
		t.Fatalf("filter not registered: %v", err)
	}

	p, err := factory(map[string]any{
		"patterns": []any{`sk-[a-zA-Z0-9]{20,}`},
	})
	if err != nil {
		t.Fatalf("create filter: %v", err)
	}

	om := p.(middleware.OutputMiddleware)
	input := "my key is sk-abcdefghijklmnopqrstuvwxyz and more"
	chunk := middleware.NewChunk([]byte(input), middleware.Output)
	result := om.ProcessOutput(chunk)

	got := string(result.Data())
	expected := "my key is [REDACTED] and more"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestFilter_MultiplePatterns(t *testing.T) {
	factory, _ := plugin.Get("filter")
	p, err := factory(map[string]any{
		"patterns": []any{`sk-[a-zA-Z0-9]+`, `AKIA[0-9A-Z]{16}`},
	})
	if err != nil {
		t.Fatal(err)
	}

	om := p.(middleware.OutputMiddleware)
	input := "openai: sk-abc123 aws: AKIA1234567890ABCDEF"
	chunk := middleware.NewChunk([]byte(input), middleware.Output)
	result := om.ProcessOutput(chunk)

	got := string(result.Data())
	expected := "openai: [REDACTED] aws: [REDACTED]"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestFilter_NoPatterns_Passthrough(t *testing.T) {
	factory, _ := plugin.Get("filter")
	p, err := factory(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	om := p.(middleware.OutputMiddleware)
	chunk := middleware.NewChunk([]byte("sk-something"), middleware.Output)
	result := om.ProcessOutput(chunk)

	if string(result.Data()) != "sk-something" {
		t.Errorf("expected passthrough, got %q", result.Data())
	}
}

func TestFilter_InvalidRegex(t *testing.T) {
	factory, _ := plugin.Get("filter")
	_, err := factory(map[string]any{
		"patterns": []any{`[invalid`},
	})
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestFilter_InputRedaction(t *testing.T) {
	factory, _ := plugin.Get("filter")
	p, err := factory(map[string]any{
		"patterns": []any{`secret-\w+`},
	})
	if err != nil {
		t.Fatal(err)
	}

	im := p.(middleware.InputMiddleware)
	chunk := middleware.NewChunk([]byte("my secret-password here"), middleware.Input)
	result := im.ProcessInput(chunk)

	got := string(result.Data())
	if got != "my [REDACTED] here" {
		t.Errorf("expected redacted input, got %q", got)
	}
}

func TestFilter_NoMatch_Passthrough(t *testing.T) {
	factory, _ := plugin.Get("filter")
	p, err := factory(map[string]any{
		"patterns": []any{`zzz-never-match`},
	})
	if err != nil {
		t.Fatal(err)
	}

	om := p.(middleware.OutputMiddleware)
	chunk := middleware.NewChunk([]byte("normal text"), middleware.Output)
	result := om.ProcessOutput(chunk)

	if string(result.Data()) != "normal text" {
		t.Errorf("expected passthrough, got %q", result.Data())
	}
}
