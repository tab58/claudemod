package middleware

import (
	"bytes"
	"strings"
	"testing"
)

// --- test helpers ---

type uppercasePlugin struct{}

func (uppercasePlugin) Name() string { return "uppercase" }
func (uppercasePlugin) ProcessInput(c Chunk) Chunk {
	return c.WithData([]byte(strings.ToUpper(string(c.Data()))))
}
func (uppercasePlugin) ProcessOutput(c Chunk) Chunk {
	return c.WithData([]byte(strings.ToUpper(string(c.Data()))))
}

type prefixPlugin struct{ prefix string }

func (p prefixPlugin) Name() string { return "prefix" }
func (p prefixPlugin) ProcessInput(c Chunk) Chunk {
	return c.WithData(append([]byte(p.prefix), c.Data()...))
}
func (p prefixPlugin) ProcessOutput(c Chunk) Chunk {
	return c.WithData(append([]byte(p.prefix), c.Data()...))
}

type dropPlugin struct{}

func (dropPlugin) Name() string                  { return "drop" }
func (dropPlugin) ProcessInput(c Chunk) Chunk     { return c.WithData(nil) }
func (dropPlugin) ProcessOutput(c Chunk) Chunk    { return c.WithData(nil) }

type outputOnlyPlugin struct{}

func (outputOnlyPlugin) Name() string               { return "output-only" }
func (outputOnlyPlugin) ProcessOutput(c Chunk) Chunk { return c.WithData([]byte("intercepted")) }

// --- tests ---

func TestPipeline_EmptyPassthrough(t *testing.T) {
	p := NewPipeline(nil)
	input := NewChunk([]byte("hello"), Input)
	output := NewChunk([]byte("world"), Output)

	resultIn := p.ProcessInput(input)
	resultOut := p.ProcessOutput(output)

	if !bytes.Equal(resultIn.Data(), []byte("hello")) {
		t.Errorf("empty pipeline modified input: got %q", resultIn.Data())
	}
	if !bytes.Equal(resultOut.Data(), []byte("world")) {
		t.Errorf("empty pipeline modified output: got %q", resultOut.Data())
	}
}

func TestPipeline_SingleMiddleware(t *testing.T) {
	p := NewPipeline([]Plugin{uppercasePlugin{}})
	chunk := NewChunk([]byte("hello"), Input)

	result := p.ProcessInput(chunk)
	if string(result.Data()) != "HELLO" {
		t.Errorf("expected 'HELLO', got %q", result.Data())
	}
}

func TestPipeline_ChainedMiddleware(t *testing.T) {
	p := NewPipeline([]Plugin{
		uppercasePlugin{},
		prefixPlugin{prefix: ">>"},
	})
	chunk := NewChunk([]byte("hello"), Input)

	result := p.ProcessInput(chunk)
	if string(result.Data()) != ">>HELLO" {
		t.Errorf("expected '>>HELLO', got %q", result.Data())
	}
}

func TestPipeline_DropStopsChain(t *testing.T) {
	p := NewPipeline([]Plugin{
		dropPlugin{},
		prefixPlugin{prefix: "should-not-reach"},
	})
	chunk := NewChunk([]byte("hello"), Input)

	result := p.ProcessInput(chunk)
	if result.Len() != 0 {
		t.Errorf("expected empty chunk after drop, got %q", result.Data())
	}
}

func TestPipeline_OutputOnly(t *testing.T) {
	p := NewPipeline([]Plugin{outputOnlyPlugin{}})

	// Should not affect input (outputOnlyPlugin doesn't implement InputMiddleware)
	inChunk := NewChunk([]byte("hello"), Input)
	result := p.ProcessInput(inChunk)
	if string(result.Data()) != "hello" {
		t.Errorf("output-only plugin affected input: got %q", result.Data())
	}

	// Should transform output
	outChunk := NewChunk([]byte("hello"), Output)
	result = p.ProcessOutput(outChunk)
	if string(result.Data()) != "intercepted" {
		t.Errorf("expected 'intercepted', got %q", result.Data())
	}
}

func TestPipeline_InputFunc_Nil_WhenEmpty(t *testing.T) {
	p := NewPipeline(nil)
	if p.InputFunc() != nil {
		t.Error("expected nil InputFunc for empty pipeline")
	}
}

func TestPipeline_OutputFunc_Nil_WhenEmpty(t *testing.T) {
	p := NewPipeline(nil)
	if p.OutputFunc() != nil {
		t.Error("expected nil OutputFunc for empty pipeline")
	}
}

func TestPipeline_InputFunc_Transforms(t *testing.T) {
	p := NewPipeline([]Plugin{uppercasePlugin{}})
	fn := p.InputFunc()
	if fn == nil {
		t.Fatal("expected non-nil InputFunc")
	}

	result := fn([]byte("hello"))
	if string(result) != "HELLO" {
		t.Errorf("expected 'HELLO', got %q", result)
	}
}

func TestPipeline_OutputFunc_Transforms(t *testing.T) {
	p := NewPipeline([]Plugin{uppercasePlugin{}})
	fn := p.OutputFunc()
	if fn == nil {
		t.Fatal("expected non-nil OutputFunc")
	}

	result := fn([]byte("hello"))
	if string(result) != "HELLO" {
		t.Errorf("expected 'HELLO', got %q", result)
	}
}
