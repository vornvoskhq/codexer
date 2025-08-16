// Package llmlog: logger.go
// Implements asynchronous structured logging for LLM calls.

package llmlog

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Logger is responsible for writing LLM call logs to a file asynchronously.
type Logger struct {
	cfg       *Config
	out       io.Writer
	file      *os.File // underlying file handle (if used)
	mu        sync.Mutex
	enabled   bool
	// TODO: Add log rotation fields (size, time, etc.)
	// TODO: Add channel/buffer for async logging.
}

// NewLogger creates a new Logger instance with the provided config.
func NewLogger(cfg *Config) (*Logger, error) {
	var out io.Writer = os.Stdout
	var file *os.File

	if cfg.Enabled && cfg.FilePath != "" {
		// Ensure parent directories exist
		dir := filepath.Dir(cfg.FilePath)
		// TODO: Handle error if directory creation fails.
		if err := os.MkdirAll(dir, 0755); err != nil {
			// Fallback to stdout on error (could also return error)
			out = os.Stdout
		} else {
			// Open file for append, create if not exists
			f, err := os.OpenFile(cfg.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				// Fallback to stdout on error
				out = os.Stdout
			} else {
				out = f
				file = f
			}
		}
	}
	return &Logger{
		cfg:     cfg,
		enabled: cfg.Enabled,
		out:     out,
		file:    file,
	}, nil
}

// Log writes a log entry asynchronously.
// entry: structured LLM log entry (to be defined).
func (l *Logger) Log(ctx context.Context, entry interface{}) error {
	if !l.enabled {
		return nil
	}
	// TODO: Serialize entry and enqueue for async write.
	return nil
}

// Rotate triggers a log file rotation (stub).
func (l *Logger) Rotate() error {
	// TODO: Implement log rotation logic.
	return nil
}

// Close flushes any pending logs and closes the log file if it's not stdout.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	// TODO: Flush async buffer/channel if implemented.
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

// EnableLogging turns logging on.
func (l *Logger) EnableLogging() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = true
}

// DisableLogging turns logging off.
func (l *Logger) DisableLogging() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = false
}

// TODO: Add graceful shutdown and flush methods for async buffer.