package middleware

// Pipeline holds ordered lists of input and output middleware.
// An empty pipeline passes data through unmodified.
type Pipeline struct {
	inputMiddleware  []InputMiddleware
	outputMiddleware []OutputMiddleware
}

// NewPipeline creates a Pipeline from the given plugins.
// Each plugin is inspected for InputMiddleware and OutputMiddleware interfaces
// and added to the appropriate chain.
func NewPipeline(plugins []Plugin) Pipeline {
	var input []InputMiddleware
	var output []OutputMiddleware

	for _, p := range plugins {
		if im, ok := p.(InputMiddleware); ok {
			input = append(input, im)
		}
		if om, ok := p.(OutputMiddleware); ok {
			output = append(output, om)
		}
	}

	return Pipeline{
		inputMiddleware:  input,
		outputMiddleware: output,
	}
}

// ProcessInput runs all input middleware on the chunk in order.
// Returns the (possibly transformed) chunk.
func (p Pipeline) ProcessInput(chunk Chunk) Chunk {
	for _, mw := range p.inputMiddleware {
		chunk = mw.ProcessInput(chunk)
		if chunk.Len() == 0 {
			return chunk
		}
	}
	return chunk
}

// ProcessOutput runs all output middleware on the chunk in order.
// Returns the (possibly transformed) chunk.
func (p Pipeline) ProcessOutput(chunk Chunk) Chunk {
	for _, mw := range p.outputMiddleware {
		chunk = mw.ProcessOutput(chunk)
		if chunk.Len() == 0 {
			return chunk
		}
	}
	return chunk
}

// InputFunc creates a ProcessInput callback suitable for bridge.Config.
func (p Pipeline) InputFunc() func([]byte) []byte {
	if len(p.inputMiddleware) == 0 {
		return nil
	}
	return func(data []byte) []byte {
		chunk := NewChunk(data, Input)
		result := p.ProcessInput(chunk)
		return result.Data()
	}
}

// OutputFunc creates a ProcessOutput callback suitable for bridge.Config.
func (p Pipeline) OutputFunc() func([]byte) []byte {
	if len(p.outputMiddleware) == 0 {
		return nil
	}
	return func(data []byte) []byte {
		chunk := NewChunk(data, Output)
		result := p.ProcessOutput(chunk)
		return result.Data()
	}
}
