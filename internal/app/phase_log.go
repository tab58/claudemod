package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const phaseLogFileName = "PHASE_LOG.jsonl"

// PhaseLogEntry represents a single entry in the append-only phase log.
type PhaseLogEntry struct {
	Timestamp         string `json:"timestamp"`
	Phase             string `json:"phase"`
	Action            string `json:"action"`
	DiscussionSummary string `json:"discussion_summary,omitempty"`
	Recommendation    string `json:"recommendation,omitempty"`
	Explanation       string `json:"explanation,omitempty"`
}

// appendPhaseLog appends a single entry to PHASE_LOG.jsonl.
func (a *App) appendPhaseLog(entry PhaseLogEntry) error {
	path := filepath.Join(a.claudeModDir, phaseLogFileName)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open phase log: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal phase log entry: %w", err)
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write phase log entry: %w", err)
	}
	return nil
}

// readPhaseLog reads all entries from PHASE_LOG.jsonl.
// Returns nil, nil if the file does not exist (first run).
func (a *App) readPhaseLog() ([]PhaseLogEntry, error) {
	path := filepath.Join(a.claudeModDir, phaseLogFileName)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open phase log: %w", err)
	}
	defer f.Close()

	var entries []PhaseLogEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max per line
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry PhaseLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("parse phase log line %d: %w", lineNum, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan phase log: %w", err)
	}
	return entries, nil
}

// formatPhaseLog renders phase log entries as a human-readable bullet list
// suitable for template injection. Returns empty string if no entries.
func formatPhaseLog(entries []PhaseLogEntry) string {
	if len(entries) == 0 {
		return ""
	}
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "- [%s] Phase '%s': action=%s", e.Timestamp, e.Phase, e.Action)
		if e.DiscussionSummary != "" {
			fmt.Fprintf(&b, " | summary: %s", e.DiscussionSummary)
		}
		if e.Explanation != "" {
			fmt.Fprintf(&b, " | explanation: %s", e.Explanation)
		}
		if e.Recommendation != "" {
			fmt.Fprintf(&b, " | recommendation: %s", e.Recommendation)
		}
		b.WriteString("\n")
	}
	return b.String()
}
