package llmlog_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"plandex-server/pkg/llmlog"
)

func TestNewLogger_FileAndStdoutFallback(t *testing.T) {
	// Create a temp dir and file path
	tempDir, err := ioutil.TempDir("", "llmlogtest")
	if err != nil {
		t.Fatalf("TempDir failed: %v", err)
	}
	defer os.RemoveAll(tempDir)
	tempFile := filepath.Join(tempDir, "logfile.log")

	cfg := &llmlog.Config{
		Enabled:  true,
		FilePath: tempFile,
	}
	logger, err := llmlog.NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	if logger == nil {
		t.Fatal("NewLogger returned nil logger")
	}
	if logger.file == nil {
		t.Errorf("Expected file handle to be non-nil (file logging), got nil")
	}

	// Close should close the file and set handle to nil
	if err := logger.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
	if logger.file != nil {
		t.Errorf("Expected file handle to be nil after Close, got non-nil")
	}
}

func TestNewLogger_InvalidDirFallbackToStdout(t *testing.T) {
	// Intentionally use a bad file path (assuming /root is unwritable by normal user)
	cfg := &llmlog.Config{
		Enabled:  true,
		FilePath: "/root/llmlog_should_fail/test.log",
	}
	logger, err := llmlog.NewLogger(cfg)
	if err != nil {
		// Should not error, should fallback to stdout
		t.Fatalf("NewLogger errored, expected fallback: %v", err)
	}
	if logger.file != nil {
		t.Errorf("Expected file handle to be nil for unwritable path, got non-nil")
	}
	// Close should not panic/error
	if err := logger.Close(); err != nil {
		t.Errorf("Close returned error on nil file: %v", err)
	}
}

func TestLogger_Close_Idempotent(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "llmlogtest")
	if err != nil {
		t.Fatalf("TempDir failed: %v", err)
	}
	defer os.RemoveAll(tempDir)
	tempFile := filepath.Join(tempDir, "logfile.log")

	cfg := &llmlog.Config{
		Enabled:  true,
		FilePath: tempFile,
	}
	logger, err := llmlog.NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("First Close failed: %v", err)
	}
	// Second close should not panic or error
	if err := logger.Close(); err != nil {
		t.Errorf("Second Close returned error: %v", err)
	}
}