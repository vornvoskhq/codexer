# Plandex LLM Call Logging Integration

This document describes the comprehensive LLM call logging integration for Plandex, including package structure, environment configuration, default file locations, and log interpretation.

---

## Modified and Added Files

### Server Integration
- `app/server/main.go`: 
  - Imports the `llmlog` package.
  - Loads logging config from environment variables.
  - Initializes a logger and registers it with the LLM client.
  - Adds shutdown hook to flush/close logger.
  - Wraps the LLM client (`model.GetLiteLLMClient()`) with `llmlog.WrapClient` (stub).

### Logging Package (`pkg/llmlog`)
- `config.go`: Configuration struct and environment loader (`LoadConfigFromEnv()`).
- `logger.go`: Asynchronous logger, file/stdout selection, graceful close.
- `wrapper.go`, `formatter.go`: Wrappers for client instrumentation and formatting (skeletons).

### Tests
- `test/llmlog/config_test.go`: Tests for environment variable configuration.
- `test/llmlog/logger_test.go`: Tests for file open, fallback behavior, and closing.

### Documentation
- `docs/llm-logging.md`: This file.

---

## Environment Variables

| Variable                              | Type    | Description                                                                                 |
|----------------------------------------|---------|---------------------------------------------------------------------------------------------|
| PLANDEX_LLM_LOGGING                    | bool    | Enable/disable LLM call logging. Default: false.                                            |
| PLANDEX_LLM_LOG_FILE                   | string  | Destination file path for logs. Default: ~/.plandex/llm_calls.log                           |
| PLANDEX_LLM_LOG_FORMAT                 | string  | Log output format: `json`, `text`, or `csv`. Default: `json`                                |
| PLANDEX_LLM_LOG_LEVEL                  | string  | Logging verbosity: `basic` or `detailed`. Default: `basic`                                  |
| PLANDEX_LLM_LOG_RETENTION              | int     | Log retention period (days). Default: 7                                                     |
| PLANDEX_LLM_LOG_MAX_FILE_SIZE          | string  | Maximum log file size (e.g. "100MB"). Default: "100MB"                                      |
| PLANDEX_LLM_LOG_INCLUDE_CONTENT        | bool    | Include prompt/response content. Default: true                                              |
| PLANDEX_LLM_LOG_INCLUDE_SENSITIVE      | bool    | Include sensitive data such as raw inputs. Default: false                                   |
| PLANDEX_LLM_LOG_FILTER_ROLES           | csv     | Comma-separated list of Plandex roles to log (e.g. "planner,builder"). Default: "*"         |
| PLANDEX_LLM_LOG_FILTER_PROVIDERS       | csv     | Comma-separated list of LLM providers to log (e.g. "openai,anthropic"). Default: "*"        |

Unset variables fall back to the documented default.

---

## Log File Location

- **Default**: `~/.plandex/llm_calls.log`
- **Override**: Set `PLANDEX_LLM_LOG_FILE` to use a custom file path.
- If directory creation or file open fails, logs are written to STDOUT as a fallback.

---

## Log Format and Interpretation

- **Format**: Configurable via `PLANDEX_LLM_LOG_FORMAT` (`json`, `text`, or `csv`).
- **Default**: JSON, one structured object per line.
- **Fields**: Each log entry contains metadata such as:
  - `timestamp`: Call time (ISO8601)
  - `request_id`, `session_id`: Unique identifiers
  - `role`, `provider`, `model`: Source and model information
  - `input`, `output`: Prompt and response content (may be redacted)
  - `limits`, `timing`, `status`, `metadata`: Usage and performance details

**Interpretation Guidelines:**
- Each line is a single LLM call ("request/response").
- Use `role`, `provider`, `model` fields to filter specific sources.
- Analyze `input.tokens`, `output.tokens` and `limits.context_window` to troubleshoot token overflows.
- Use timestamps and durations for performance analysis.

See the code in `pkg/llmlog/formatter.go` for examples of formatting. Log content is controlled by the configuration and can be filtered at source or during analysis.

---