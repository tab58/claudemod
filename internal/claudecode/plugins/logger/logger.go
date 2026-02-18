package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/tab58/claudemod/internal/claudecode/ansi"
	"github.com/tab58/claudemod/internal/claudecode/config"
	"github.com/tab58/claudemod/internal/claudecode/middleware"
	"github.com/tab58/claudemod/internal/claudecode/plugin"
)

func init() {
	plugin.Register("logger", newLogger)
}

// entry is a single JSONL log record.
type entry struct {
	Timestamp string `json:"timestamp"`
	SessionID string `json:"session_id"`
	Direction string `json:"direction"`
	Data      string `json:"data"`
	RawLen    int    `json:"raw_len"`
}

// Logger is an output middleware that writes JSONL audit logs.
// It reads but never modifies the data stream.
type Logger struct {
	mu        sync.Mutex
	file      *os.File
	sessionID string
	logInput  bool
	logOutput bool
}

func newLogger(opts map[string]any) (middleware.Plugin, error) {
	logDir := "~/.claudemod/logs"
	if dir, ok := opts["log_dir"].(string); ok && dir != "" {
		logDir = dir
	}
	logDir = config.ExpandHome(logDir)

	if err := os.MkdirAll(logDir, 0750); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	sessionID := uuid.New().String()
	filename := filepath.Join(logDir, fmt.Sprintf("session-%s.jsonl", sessionID))

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	logInput := true
	if v, ok := opts["log_input"].(bool); ok {
		logInput = v
	}
	logOutput := true
	if v, ok := opts["log_output"].(bool); ok {
		logOutput = v
	}

	return &Logger{
		file:      f,
		sessionID: sessionID,
		logInput:  logInput,
		logOutput: logOutput,
	}, nil
}

func (l *Logger) Name() string { return "logger" }

// Close flushes and closes the log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// ProcessInput logs input chunks without modifying them.
func (l *Logger) ProcessInput(chunk middleware.Chunk) middleware.Chunk {
	if l.logInput {
		l.writeEntry("input", chunk)
	}
	return chunk
}

// ProcessOutput logs output chunks without modifying them.
func (l *Logger) ProcessOutput(chunk middleware.Chunk) middleware.Chunk {
	if l.logOutput {
		l.writeEntry("output", chunk)
	}
	return chunk
}

func (l *Logger) writeEntry(direction string, chunk middleware.Chunk) {
	data := chunk.Data()
	clean := string(ansi.Strip(data))

	e := entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		SessionID: l.sessionID,
		Direction: direction,
		Data:      clean,
		RawLen:    len(data),
	}

	line, err := json.Marshal(e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: marshal error: %v\n", err)
		return
	}
	line = append(line, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.file.Write(line)
}
