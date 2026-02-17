package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbright/claudemod/internal/middleware"
	"github.com/tbright/claudemod/internal/plugin"
)

func TestLogger_CreatesLogFile(t *testing.T) {
	dir := t.TempDir()
	factory, err := plugin.Get("logger")
	if err != nil {
		t.Fatalf("logger not registered: %v", err)
	}

	p, err := factory(map[string]any{"log_dir": dir})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	// Process a chunk
	om := p.(middleware.OutputMiddleware)
	chunk := middleware.NewChunk([]byte("hello"), middleware.Output)
	result := om.ProcessOutput(chunk)

	// Data should pass through unmodified
	if string(result.Data()) != "hello" {
		t.Errorf("logger modified data: got %q", result.Data())
	}

	// Check log file was created
	matches, _ := filepath.Glob(filepath.Join(dir, "session-*.jsonl"))
	if len(matches) != 1 {
		t.Fatalf("expected 1 log file, got %d", len(matches))
	}

	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}

	var e entry
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &e); err != nil {
		t.Fatalf("invalid JSONL: %v", err)
	}
	if e.Direction != "output" {
		t.Errorf("expected direction 'output', got %q", e.Direction)
	}
	if e.Data != "hello" {
		t.Errorf("expected data 'hello', got %q", e.Data)
	}
	if e.RawLen != 5 {
		t.Errorf("expected raw_len 5, got %d", e.RawLen)
	}
}

func TestLogger_StripsANSI(t *testing.T) {
	dir := t.TempDir()
	factory, _ := plugin.Get("logger")
	p, err := factory(map[string]any{"log_dir": dir})
	if err != nil {
		t.Fatal(err)
	}

	om := p.(middleware.OutputMiddleware)
	chunk := middleware.NewChunk([]byte("\x1b[31mred\x1b[0m"), middleware.Output)
	om.ProcessOutput(chunk)

	matches, _ := filepath.Glob(filepath.Join(dir, "session-*.jsonl"))
	data, _ := os.ReadFile(matches[0])

	var e entry
	json.Unmarshal([]byte(strings.TrimSpace(string(data))), &e)
	if e.Data != "red" {
		t.Errorf("expected ANSI-stripped 'red', got %q", e.Data)
	}
}

func TestLogger_InputLogging(t *testing.T) {
	dir := t.TempDir()
	factory, _ := plugin.Get("logger")
	p, err := factory(map[string]any{"log_dir": dir, "log_input": true})
	if err != nil {
		t.Fatal(err)
	}

	im := p.(middleware.InputMiddleware)
	chunk := middleware.NewChunk([]byte("typed"), middleware.Input)
	result := im.ProcessInput(chunk)

	if string(result.Data()) != "typed" {
		t.Errorf("logger modified input data: got %q", result.Data())
	}

	matches, _ := filepath.Glob(filepath.Join(dir, "session-*.jsonl"))
	data, _ := os.ReadFile(matches[0])

	var e entry
	json.Unmarshal([]byte(strings.TrimSpace(string(data))), &e)
	if e.Direction != "input" {
		t.Errorf("expected direction 'input', got %q", e.Direction)
	}
}

func TestLogger_DisabledInput(t *testing.T) {
	dir := t.TempDir()
	factory, _ := plugin.Get("logger")
	p, err := factory(map[string]any{"log_dir": dir, "log_input": false, "log_output": false})
	if err != nil {
		t.Fatal(err)
	}

	im := p.(middleware.InputMiddleware)
	om := p.(middleware.OutputMiddleware)

	im.ProcessInput(middleware.NewChunk([]byte("in"), middleware.Input))
	om.ProcessOutput(middleware.NewChunk([]byte("out"), middleware.Output))

	matches, _ := filepath.Glob(filepath.Join(dir, "session-*.jsonl"))
	data, _ := os.ReadFile(matches[0])
	if len(strings.TrimSpace(string(data))) != 0 {
		t.Error("expected empty log when input and output logging disabled")
	}
}
