// Package llmlog provides structured logging for LLM API calls.
package llmlog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Logger is responsible for writing LLM call logs to a file asynchronously.
type Logger struct {
	cfg     *Config
	out     io.Writer
	file    *os.File // underlying file handle (if used)
	mu      sync.Mutex
	enabled *atomic.Bool // Use atomic for thread-safe enabled/disabled state
	closed  *atomic.Bool // Track if logger is closed
	wg      sync.WaitGroup // Wait group for async operations
}

// NewLogger creates a new Logger instance with the provided config.
func NewLogger(cfg *Config) (*Logger, error) {
	var out io.Writer = os.Stdout
	var file *os.File

	if cfg.Enabled && cfg.FilePath != "" {
		// Ensure parent directories exist
		dir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory %s: %w", dir, err)
		}

		// Open file for append, create if not exists
		f, err := os.OpenFile(cfg.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", cfg.FilePath, err)
		}

		out = f
		file = f
	}

	enabled := &atomic.Bool{}
	enabled.Store(cfg.Enabled)

	closed := &atomic.Bool{}
	closed.Store(false)

	return &Logger{
		cfg:     cfg,
		enabled: enabled,
		closed:  closed,
		out:     out,
		file:    file,
	}, nil
}

// LogRequest logs an LLM API request
func (l *Logger) LogRequest(ctx context.Context, req *LLMRequest) error {
	if !l.enabled.Load() || l.closed.Load() {
		return nil
	}

	// Create a structured log entry
	logEntry := map[string]interface{}{
		"timestamp":   req.Timestamp.UTC().Format(time.RFC3339Nano),
		"request_id":  req.RequestID,
		"model":       req.Model,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"provider":    req.Provider,
		"type":        "request",
	}

	// Include all messages in the log
	if len(req.Messages) > 0 {
		messages := make([]map[string]string, 0, len(req.Messages))
		for _, msg := range req.Messages {
			messages = append(messages, map[string]string{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
		logEntry["messages"] = messages

		// For backward compatibility, also log the last message as "prompt"
		logEntry["prompt"] = req.Messages[len(req.Messages)-1].Content
	}

	return l.logJSON(logEntry)
}

// LogStreamChunk logs a chunk of streaming response
func (l *Logger) LogStreamChunk(requestID string, chunk string) error {
	if !l.enabled.Load() || l.closed.Load() || chunk == "" {
		return nil
	}

	logEntry := map[string]interface{}{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"request_id": requestID,
		"chunk":      chunk,
		"type":       "stream_chunk",
	}

	return l.logJSON(logEntry)
}

// LogStreamEnd logs the end of a streaming response
func (l *Logger) LogStreamEnd(requestID string, err error) error {
	if !l.enabled.Load() || l.closed.Load() {
		return nil
	}

	logEntry := map[string]interface{}{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"request_id": requestID,
		"type":       "stream_end",
	}

	if err != nil {
		logEntry["error"] = err.Error()
	}

	return l.logJSON(logEntry)
}

// LogError logs an error that occurred during an LLM API call
func (l *Logger) LogError(ctx context.Context, requestID string, err error) error {
	if err == nil || !l.enabled.Load() || l.closed.Load() {
		return nil
	}

	logEntry := map[string]interface{}{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"request_id": requestID,
		"type":       "error",
		"error":      err.Error(),
	}

	return l.logJSON(logEntry)
}

// logJSON is a helper function to log JSON-encoded data
func (l *Logger) logJSON(data map[string]interface{}) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed.Load() {
		return fmt.Errorf("logger is closed")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// Add newline for readability
	jsonData = append(jsonData, '\n')

	_, err = l.out.Write(jsonData)
	if err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	return nil
}

// Log is kept for backward compatibility
func (l *Logger) Log(ctx context.Context, entry interface{}) error {
	if !l.enabled.Load() || l.closed.Load() {
		return nil
	}

	switch v := entry.(type) {
	case *LLMRequest:
		return l.LogRequest(ctx, v)
	case error:
		return l.LogError(ctx, "", v)
	default:
		// Try to log as JSON if it's not a known type
		logEntry := map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"data":      entry,
			"type":      "log",
		}
		return l.logJSON(logEntry)
	}
}

// Rotate triggers a log file rotation.
func (l *Logger) Rotate() error {
	if l.file == nil || l.file == os.Stdout || l.closed.Load() {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Close the current file
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	// Rename the current log file with a timestamp
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", l.cfg.FilePath, timestamp)
	if err := os.Rename(l.cfg.FilePath, rotatedPath); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	// Reopen the log file
	f, err := os.OpenFile(l.cfg.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to reopen log file: %w", err)
	}

	l.file = f
	l.out = f

	return nil
}

// Close flushes any pending logs and closes the log file if it's not stdout.
func (l *Logger) Close() error {
	if l.closed.Load() {
		return nil // Already closed
	}

	// Mark as closed first to prevent new log entries
	l.closed.Store(true)
	l.enabled.Store(false)

	l.mu.Lock()
	defer l.mu.Unlock()

	// Close the file if it's not stdout
	if l.file != nil && l.file != os.Stdout {
		if err := l.file.Sync(); err != nil {
			return fmt.Errorf("failed to sync log file: %w", err)
		}
		if err := l.file.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
	}

	return nil
}

// EnableLogging turns logging on.
func (l *Logger) EnableLogging() {
	if !l.closed.Load() {
		l.enabled.Store(true)
	}
}

// DisableLogging turns logging off.
func (l *Logger) DisableLogging() {
	l.enabled.Store(false)
}

// IsEnabled returns true if logging is currently enabled.
func (l *Logger) IsEnabled() bool {
	return l.enabled.Load() && !l.closed.Load()
}

// IsClosed returns true if the logger has been closed.
func (l *Logger) IsClosed() bool {
	return l.closed.Load()
}