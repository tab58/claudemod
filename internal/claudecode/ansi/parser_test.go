package ansi

import "testing"

func TestStrip(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain text", "hello world", "hello world"},
		{"color code", "\x1b[31mred text\x1b[0m", "red text"},
		{"bold", "\x1b[1mbold\x1b[22m", "bold"},
		{"cursor move", "\x1b[2Jhello", "hello"},
		{"multiple sequences", "\x1b[31m\x1b[1mhello\x1b[0m world", "hello world"},
		{"osc title", "\x1b]0;My Title\x07content", "content"},
		{"osc with st", "\x1b]0;Title\x1b\\content", "content"},
		{"empty", "", ""},
		{"no escapes", "plain text 123", "plain text 123"},
		{"sgr with params", "\x1b[38;5;196mcolored\x1b[0m", "colored"},
		{"mixed content", "before\x1b[32mgreen\x1b[0mafter", "beforegreenafter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(Strip([]byte(tt.input)))
			if got != tt.expected {
				t.Errorf("Strip(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStripString(t *testing.T) {
	input := "\x1b[31mhello\x1b[0m"
	got := StripString(input)
	if got != "hello" {
		t.Errorf("StripString(%q) = %q, want %q", input, got, "hello")
	}
}
