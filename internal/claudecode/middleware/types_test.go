package middleware

import (
	"testing"
)

func TestNewChunk_CopiesData(t *testing.T) {
	original := []byte("hello")
	chunk := NewChunk(original, Input)

	// Mutating original should not affect chunk
	original[0] = 'X'
	if chunk.Data()[0] == 'X' {
		t.Error("NewChunk did not copy data — mutation leaked")
	}
}

func TestChunk_DataReturnsCopy(t *testing.T) {
	chunk := NewChunk([]byte("hello"), Output)
	data := chunk.Data()
	data[0] = 'X'

	if chunk.Data()[0] == 'X' {
		t.Error("Data() did not return a copy — mutation leaked")
	}
}

func TestChunk_Direction(t *testing.T) {
	tests := []struct {
		name string
		dir  Direction
	}{
		{"input", Input},
		{"output", Output},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := NewChunk([]byte("test"), tt.dir)
			if chunk.Direction() != tt.dir {
				t.Errorf("expected direction %d, got %d", tt.dir, chunk.Direction())
			}
		})
	}
}

func TestChunk_Len(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{"empty", []byte{}, 0},
		{"non-empty", []byte("hello"), 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := NewChunk(tt.data, Input)
			if chunk.Len() != tt.expected {
				t.Errorf("expected len %d, got %d", tt.expected, chunk.Len())
			}
		})
	}
}

func TestChunk_WithData(t *testing.T) {
	original := NewChunk([]byte("hello"), Input)
	modified := original.WithData([]byte("world"))

	if string(original.Data()) != "hello" {
		t.Error("WithData mutated the original chunk")
	}
	if string(modified.Data()) != "world" {
		t.Errorf("expected 'world', got '%s'", modified.Data())
	}
	if modified.Direction() != original.Direction() {
		t.Error("WithData changed direction")
	}
	if modified.Timestamp() != original.Timestamp() {
		t.Error("WithData changed timestamp")
	}
}

func TestChunk_WithData_CopiesInput(t *testing.T) {
	chunk := NewChunk([]byte("hello"), Input)
	newData := []byte("world")
	modified := chunk.WithData(newData)

	newData[0] = 'X'
	if modified.Data()[0] == 'X' {
		t.Error("WithData did not copy data — mutation leaked")
	}
}
