package llmlog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_LogRequest(t *testing.T) {
	// Create a temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "llmlog-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize logger with a test log file
	logPath := filepath.Join(tempDir, "test.log")
	logger, err := NewLogger(&Config{
		Enabled:  true,
		FilePath: logPath,
	})
	require.NoError(t, err)
	defer logger.Close()

	// Create a test request
	req := &LLMRequest{
		RequestID:   "test-request-123",
		Model:       "test-model",
		Messages:    []openai.ChatCompletionMessage{{Role: "user", Content: "Hello, world!"}},
		Temperature: 0.7,
		MaxTokens:   100,
		Provider:    "test-provider",
		Timestamp:   time.Now(),
	}

	// Log the request
	err = logger.LogRequest(context.Background(), req)
	require.NoError(t, err)

	// Read the log file
	logData, err := os.ReadFile(logPath)
	require.NoError(t, err)

	// Verify the log entry
	logStr := string(logData)
	assert.Contains(t, logStr, `"request_id":"test-request-123"`)
	assert.Contains(t, logStr, `"model":"test-model"`)
	assert.Contains(t, logStr, `"type":"request"`)
}

func TestLogger_LogStreamChunk(t *testing.T) {
	// Create a temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "llmlog-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize logger with a test log file
	logPath := filepath.Join(tempDir, "test.log")
	logger, err := NewLogger(&Config{
		Enabled:  true,
		FilePath: logPath,
	})
	require.NoError(t, err)
	defer logger.Close()

	// Log a stream chunk
	requestID := "test-request-123"
	chunk := "Hello, world!"

	err = logger.LogStreamChunk(requestID, chunk)
	require.NoError(t, err)

	// Read the log file
	logData, err := os.ReadFile(logPath)
	require.NoError(t, err)

	// Verify the log entry
	logStr := string(logData)
	assert.Contains(t, logStr, `"request_id":"test-request-123"`)
	assert.Contains(t, logStr, `"chunk":"Hello, world!"`)
	assert.Contains(t, logStr, `"type":"stream_chunk"`)
}

func TestLogger_LogError(t *testing.T) {
	// Create a temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "llmlog-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize logger with a test log file
	logPath := filepath.Join(tempDir, "test.log")
	logger, err := NewLogger(&Config{
		Enabled:  true,
		FilePath: logPath,
	})
	require.NoError(t, err)
	defer logger.Close()

	// Log an error
	err = logger.LogError(context.Background(), "test-request-123", assert.AnError)
	require.NoError(t, err)

	// Read the log file
	logData, err := os.ReadFile(logPath)
	require.NoError(t, err)

	// Verify the log entry
	logStr := string(logData)
	assert.Contains(t, logStr, `"request_id":"test-request-123"`)
	assert.Contains(t, logStr, `"type":"error"`)
	assert.Contains(t, logStr, `"error":"assert.AnError general error for testing"`)
}

func TestLogger_Close(t *testing.T) {
	// Create a temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "llmlog-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize logger with a test log file
	logPath := filepath.Join(tempDir, "test.log")
	logger, err := NewLogger(&Config{
		Enabled:  true,
		FilePath: logPath,
	})
	require.NoError(t, err)

	// Log a message
	err = logger.LogRequest(context.Background(), &LLMRequest{
		RequestID:   "test-request-123",
		Model:       "test-model",
		Messages:    []openai.ChatCompletionMessage{},
		Temperature: 0.7,
		MaxTokens:   100,
		Provider:    "test-provider",
		Timestamp:   time.Now(),
	})
	require.NoError(t, err)

	// Close the logger
	err = logger.Close()
	require.NoError(t, err)

	// Verify the logger is closed
	assert.True(t, logger.IsClosed())

	// Try to log after closing - should not panic or return an error
	err = logger.LogRequest(context.Background(), &LLMRequest{
		RequestID:   "test-request-456",
		Model:       "test-model",
		Messages:    []openai.ChatCompletionMessage{},
		Temperature: 0.7,
		MaxTokens:   100,
		Provider:    "test-provider",
		Timestamp:   time.Now(),
	})
	// We don't return an error for logging after close, we just ignore it
	assert.NoError(t, err)
}

func TestLogger_LogRotation(t *testing.T) {
	// Create a temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "llmlog-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize logger with a test log file
	logPath := filepath.Join(tempDir, "test.log")
	logger, err := NewLogger(&Config{
		Enabled:  true,
		FilePath: logPath,
	})
	require.NoError(t, err)
	defer logger.Close()

	// Log a message
	err = logger.LogRequest(context.Background(), &LLMRequest{
		RequestID: "test-request-123",
		Model:     "test-model",
	})
	require.NoError(t, err)

	// Rotate the log file
	err = logger.Rotate()
	require.NoError(t, err)

	// Log another message
	err = logger.LogRequest(context.Background(), &LLMRequest{
		RequestID: "test-request-456",
		Model:     "test-model",
	})
	require.NoError(t, err)

	// Check that both log files exist
	logFiles, err := filepath.Glob(filepath.Join(tempDir, "test.log*"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(logFiles), 2, "Expected at least 2 log files after rotation")
}

func TestLogger_ConcurrentAccess(t *testing.T) {
	// Create a temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "llmlog-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize logger with a test log file
	logPath := filepath.Join(tempDir, "test.log")
	logger, err := NewLogger(&Config{
		Enabled:  true,
		FilePath: logPath,
	})
	require.NoError(t, err)
	defer logger.Close()

	// Number of concurrent goroutines
	numRoutines := 10
	numLogsPerRoutine := 10

	// Channel to collect errors from goroutines
	errChan := make(chan error, numRoutines*numLogsPerRoutine)

	// Start multiple goroutines that log concurrently
	for i := 0; i < numRoutines; i++ {
		go func(routineID int) {
			for j := 0; j < numLogsPerRoutine; j++ {
				reqID := fmt.Sprintf("routine-%d-log-%d", routineID, j)
				err := logger.LogRequest(context.Background(), &LLMRequest{
					RequestID:   reqID,
					Model:       "test-model",
					Messages:    []openai.ChatCompletionMessage{},
					Temperature: 0.7,
					MaxTokens:   100,
					Provider:    "test-provider",
					Timestamp:   time.Now(),
				})
				if err != nil {
					errChan <- fmt.Errorf("routine %d, log %d: %w", routineID, j, err)
				} else {
					errChan <- nil
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete and check for errors
	for i := 0; i < numRoutines*numLogsPerRoutine; i++ {
		err := <-errChan
		assert.NoError(t, err, "Error in logging goroutine")
	}

	// Close the logger to ensure all logs are flushed
	err = logger.Close()
	require.NoError(t, err)

	// Read the log file and verify all logs are present
	logData, err := os.ReadFile(logPath)
	require.NoError(t, err)

	logStr := string(logData)

	// Verify that logs from all routines are present
	for i := 0; i < numRoutines; i++ {
		expectedLog := fmt.Sprintf(`"request_id":"routine-%d-log-0"`, i)
		assert.Contains(t, logStr, expectedLog, "Missing log from routine %d", i)
	}
}
