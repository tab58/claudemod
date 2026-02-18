package ansi

import "regexp"

// ansiPattern matches ANSI escape sequences:
// - CSI sequences:  ESC [ ... final_byte
// - OSC sequences:  ESC ] ... ST (BEL or ESC \)
// - Simple escapes: ESC followed by one character
var ansiPattern = regexp.MustCompile(`\x1b(?:\[[0-9;?]*[A-Za-z]|\][^\x07\x1b]*(?:\x07|\x1b\\)|[^[\]])`)

// Strip removes all ANSI escape sequences from the input,
// returning clean text suitable for logging.
func Strip(data []byte) []byte {
	return ansiPattern.ReplaceAll(data, nil)
}

// StripString is a convenience wrapper for string input.
func StripString(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}
