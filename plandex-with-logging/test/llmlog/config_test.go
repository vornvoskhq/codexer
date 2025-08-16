package llmlog_test

import (
	"os"
	"strings"
	"testing"

	"plandex-server/pkg/llmlog"
)

func TestLoadConfigFromEnv_Defaults(t *testing.T) {
	// Clear all relevant env vars
	clearLLMLogEnv()
	cfg := llmlog.LoadConfigFromEnv()
	if cfg.Enabled != false {
		t.Errorf("Expected Enabled=false, got %v", cfg.Enabled)
	}
	if !strings.HasSuffix(cfg.FilePath, ".plandex/llm_calls.log") {
		t.Errorf("Expected default log file path to end with .plandex/llm_calls.log, got %s", cfg.FilePath)
	}
	if cfg.Format != "json" {
		t.Errorf("Expected default Format=json, got %s", cfg.Format)
	}
	if cfg.Level != "basic" {
		t.Errorf("Expected default Level=basic, got %s", cfg.Level)
	}
	if cfg.RetentionDays != 7 {
		t.Errorf("Expected default RetentionDays=7, got %d", cfg.RetentionDays)
	}
	if cfg.MaxFileSize != "100MB" {
		t.Errorf("Expected default MaxFileSize=100MB, got %s", cfg.MaxFileSize)
	}
	if cfg.IncludeContent != true {
		t.Errorf("Expected default IncludeContent=true, got %v", cfg.IncludeContent)
	}
	if cfg.IncludeSensitive != false {
		t.Errorf("Expected default IncludeSensitive=false, got %v", cfg.IncludeSensitive)
	}
	if len(cfg.Filters.Roles) != 1 || cfg.Filters.Roles[0] != "*" {
		t.Errorf("Expected default Roles = [*], got %v", cfg.Filters.Roles)
	}
	if len(cfg.Filters.Providers) != 1 || cfg.Filters.Providers[0] != "*" {
		t.Errorf("Expected default Providers = [*], got %v", cfg.Filters.Providers)
	}
}

func TestLoadConfigFromEnv_OverrideVars(t *testing.T) {
	clearLLMLogEnv()
	os.Setenv("PLANDEX_LLM_LOGGING", "true")
	os.Setenv("PLANDEX_LLM_LOG_FILE", "/tmp/test.log")
	os.Setenv("PLANDEX_LLM_LOG_FORMAT", "text")
	os.Setenv("PLANDEX_LLM_LOG_LEVEL", "detailed")
	os.Setenv("PLANDEX_LLM_LOG_RETENTION", "30")
	os.Setenv("PLANDEX_LLM_LOG_MAX_FILE_SIZE", "5MB")
	os.Setenv("PLANDEX_LLM_LOG_INCLUDE_CONTENT", "false")
	os.Setenv("PLANDEX_LLM_LOG_INCLUDE_SENSITIVE", "true")
	os.Setenv("PLANDEX_LLM_LOG_FILTER_ROLES", "planner,builder")
	os.Setenv("PLANDEX_LLM_LOG_FILTER_PROVIDERS", "openai,anthropic")

	cfg := llmlog.LoadConfigFromEnv()
	if cfg.Enabled != true {
		t.Errorf("Expected Enabled=true, got %v", cfg.Enabled)
	}
	if cfg.FilePath != "/tmp/test.log" {
		t.Errorf("Expected FilePath=/tmp/test.log, got %s", cfg.FilePath)
	}
	if cfg.Format != "text" {
		t.Errorf("Expected Format=text, got %s", cfg.Format)
	}
	if cfg.Level != "detailed" {
		t.Errorf("Expected Level=detailed, got %s", cfg.Level)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("Expected RetentionDays=30, got %d", cfg.RetentionDays)
	}
	if cfg.MaxFileSize != "5MB" {
		t.Errorf("Expected MaxFileSize=5MB, got %s", cfg.MaxFileSize)
	}
	if cfg.IncludeContent != false {
		t.Errorf("Expected IncludeContent=false, got %v", cfg.IncludeContent)
	}
	if cfg.IncludeSensitive != true {
		t.Errorf("Expected IncludeSensitive=true, got %v", cfg.IncludeSensitive)
	}
	if len(cfg.Filters.Roles) != 2 || cfg.Filters.Roles[0] != "planner" || cfg.Filters.Roles[1] != "builder" {
		t.Errorf("Expected Roles = [planner builder], got %v", cfg.Filters.Roles)
	}
	if len(cfg.Filters.Providers) != 2 || cfg.Filters.Providers[0] != "openai" || cfg.Filters.Providers[1] != "anthropic" {
		t.Errorf("Expected Providers = [openai anthropic], got %v", cfg.Filters.Providers)
	}
}

func clearLLMLogEnv() {
	os.Unsetenv("PLANDEX_LLM_LOGGING")
	os.Unsetenv("PLANDEX_LLM_LOG_FILE")
	os.Unsetenv("PLANDEX_LLM_LOG_FORMAT")
	os.Unsetenv("PLANDEX_LLM_LOG_LEVEL")
	os.Unsetenv("PLANDEX_LLM_LOG_RETENTION")
	os.Unsetenv("PLANDEX_LLM_LOG_MAX_FILE_SIZE")
	os.Unsetenv("PLANDEX_LLM_LOG_INCLUDE_CONTENT")
	os.Unsetenv("PLANDEX_LLM_LOG_INCLUDE_SENSITIVE")
	os.Unsetenv("PLANDEX_LLM_LOG_FILTER_ROLES")
	os.Unsetenv("PLANDEX_LLM_LOG_FILTER_PROVIDERS")
}