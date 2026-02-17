package inject

import (
	"sync"

	"github.com/tbright/claudemod/internal/middleware"
	"github.com/tbright/claudemod/internal/plugin"
)

func init() {
	plugin.Register("inject", newInject)
}

// Inject is an input middleware that prepends configured text to the first
// input chunk, then passes through all subsequent chunks unmodified.
type Inject struct {
	text string
	once sync.Once
}

func newInject(opts map[string]any) (middleware.Plugin, error) {
	text := ""
	if t, ok := opts["text"].(string); ok {
		text = t
	}
	return &Inject{text: text}, nil
}

func (j *Inject) Name() string { return "inject" }

// ProcessInput prepends the configured text to the first non-empty input chunk.
func (j *Inject) ProcessInput(chunk middleware.Chunk) middleware.Chunk {
	if j.text == "" {
		return chunk
	}

	var injected middleware.Chunk
	j.once.Do(func() {
		combined := append([]byte(j.text), chunk.Data()...)
		injected = chunk.WithData(combined)
	})

	if injected.Len() > 0 {
		return injected
	}
	return chunk
}
