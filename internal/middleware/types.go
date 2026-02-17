package middleware

import (
	"io"
	"time"
)

// Direction indicates whether data is flowing from user to child (Input)
// or from child to user (Output).
type Direction int

const (
	Input  Direction = iota // user -> claude
	Output                  // claude -> user
)

// Chunk is an immutable data unit flowing through the middleware pipeline.
// Middleware must never modify a Chunk — always return a new one via helper methods.
type Chunk struct {
	data      []byte
	direction Direction
	timestamp time.Time
}

// NewChunk creates a new Chunk with a copy of the provided data.
func NewChunk(data []byte, dir Direction) Chunk {
	cp := make([]byte, len(data))
	copy(cp, data)
	return Chunk{
		data:      cp,
		direction: dir,
		timestamp: time.Now(),
	}
}

// Data returns a copy of the chunk's bytes.
func (c Chunk) Data() []byte {
	cp := make([]byte, len(c.data))
	copy(cp, c.data)
	return cp
}

// Direction returns the chunk's flow direction.
func (c Chunk) Direction() Direction {
	return c.direction
}

// Timestamp returns when the chunk was created.
func (c Chunk) Timestamp() time.Time {
	return c.timestamp
}

// Len returns the byte length of the chunk data.
func (c Chunk) Len() int {
	return len(c.data)
}

// WithData returns a new Chunk with replaced data, preserving direction and timestamp.
func (c Chunk) WithData(data []byte) Chunk {
	cp := make([]byte, len(data))
	copy(cp, data)
	return Chunk{
		data:      cp,
		direction: c.direction,
		timestamp: c.timestamp,
	}
}

// InputMiddleware processes chunks flowing from user to child.
type InputMiddleware interface {
	ProcessInput(chunk Chunk) Chunk
}

// OutputMiddleware processes chunks flowing from child to user.
type OutputMiddleware interface {
	ProcessOutput(chunk Chunk) Chunk
}

// Plugin is the combined interface for middleware that can process both directions.
// Plugins may implement one or both of InputMiddleware and OutputMiddleware.
// Plugins that hold resources should also implement io.Closer.
type Plugin interface {
	Name() string
}

// CloseAll closes any plugins that implement io.Closer.
func CloseAll(plugins []Plugin) {
	for _, p := range plugins {
		if c, ok := p.(io.Closer); ok {
			_ = c.Close()
		}
	}
}
