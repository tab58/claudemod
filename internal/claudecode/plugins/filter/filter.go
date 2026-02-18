package filter

import (
	"fmt"
	"regexp"

	"github.com/tab58/claudemod/internal/claudecode/middleware"
	"github.com/tab58/claudemod/internal/claudecode/plugin"
)

func init() {
	plugin.Register("filter", newFilter)
}

const redactedText = "[REDACTED]"

// Filter is an output middleware that redacts sensitive patterns from data.
// It only modifies the data stream when a pattern matches.
type Filter struct {
	patterns []*regexp.Regexp
}

func newFilter(opts map[string]any) (middleware.Plugin, error) {
	rawPatterns, ok := opts["patterns"]
	if !ok {
		return &Filter{}, nil
	}

	patternList, ok := rawPatterns.([]any)
	if !ok {
		return nil, fmt.Errorf("filter: patterns must be a list of strings")
	}

	var compiled []*regexp.Regexp
	for _, raw := range patternList {
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("filter: each pattern must be a string")
		}
		re, err := regexp.Compile(s)
		if err != nil {
			return nil, fmt.Errorf("filter: invalid regex %q: %w", s, err)
		}
		compiled = append(compiled, re)
	}

	return &Filter{patterns: compiled}, nil
}

func (f *Filter) Name() string { return "filter" }

// ProcessOutput redacts matching patterns from the output data.
func (f *Filter) ProcessOutput(chunk middleware.Chunk) middleware.Chunk {
	return f.redact(chunk)
}

// ProcessInput redacts matching patterns from the input data.
func (f *Filter) ProcessInput(chunk middleware.Chunk) middleware.Chunk {
	return f.redact(chunk)
}

func (f *Filter) redact(chunk middleware.Chunk) middleware.Chunk {
	if len(f.patterns) == 0 {
		return chunk
	}

	data := chunk.Data()
	for _, re := range f.patterns {
		data = re.ReplaceAll(data, []byte(redactedText))
	}
	return chunk.WithData(data)
}
