# Plandex LLM Call Logging - Product Requirements Document

## Product Overview

### Purpose
Add comprehensive logging capabilities to Plandex to capture detailed information about all Large Language Model (LLM) interactions, enabling developers to troubleshoot "max token reached" errors and optimize LLM usage patterns.

### Problem Statement
Plandex users encounter "max token reached before conversation starts" messages without sufficient visibility into:
- Which role (planner, architect, etc.) is consuming tokens
- Token usage patterns across different models
- Request/response sizes that lead to limit violations
- Performance characteristics of LLM calls

### Solution Statement
Implement a non-destructive logging system that captures comprehensive LLM call metadata while maintaining full backward compatibility and having zero performance impact when disabled.

## Functional Requirements

### FR-1: LLM Call Capture
**Requirement**: The system must log every LLM API call made by Plandex
- **FR-1.1**: Capture calls regardless of LLM provider (OpenAI, Anthropic, etc.)
- **FR-1.2**: Log calls from all Plandex roles (planner, architect, builder, etc.)
- **FR-1.3**: Record both successful and failed LLM interactions
- **FR-1.4**: Maintain chronological order of all calls within a session

### FR-2: Comprehensive Metadata Collection
**Requirement**: Each logged LLM call must include detailed metadata
- **FR-2.1**: Timestamp (ISO 8601 format with timezone)
- **FR-2.2**: Plandex role making the call (planner, architect, builder, etc.)
- **FR-2.3**: LLM model identifier (gpt-4, claude-3-sonnet, etc.)
- **FR-2.4**: LLM provider (openai, anthropic, etc.)
- **FR-2.5**: Complete input message/prompt content
- **FR-2.6**: Complete response content
- **FR-2.7**: Input token count (actual or estimated)
- **FR-2.8**: Output token count (actual or estimated)
- **FR-2.9**: Model token limit
- **FR-2.10**: Request duration in milliseconds
- **FR-2.11**: Success/failure status
- **FR-2.12**: Error message for failed calls
- **FR-2.13**: Unique request identifier
- **FR-2.14**: Plandex session identifier
- **FR-2.15**: Additional context (temperature, max_tokens, etc.)

### FR-3: Token Counting Accuracy
**Requirement**: Provide accurate token counts for troubleshooting
- **FR-3.1**: Use provider-specific token counting when available
- **FR-3.2**: Implement tiktoken for OpenAI models
- **FR-3.3**: Use character-based estimation for unsupported models
- **FR-3.4**: Log both estimated and actual token counts when both available
- **FR-3.5**: Include token limit information for context

### FR-4: Configuration Management
**Requirement**: Provide flexible configuration options
- **FR-4.1**: Enable/disable logging globally
- **FR-4.2**: Configure log file location
- **FR-4.3**: Select output format (JSON, structured text)
- **FR-4.4**: Set logging verbosity level (basic, detailed)
- **FR-4.5**: Configure log retention period
- **FR-4.6**: Filter logging by role or provider
- **FR-4.7**: Support both configuration file and environment variable settings

### FR-5: Log Output Formats
**Requirement**: Support multiple output formats for different use cases
- **FR-5.1**: JSON format for programmatic analysis
- **FR-5.2**: Human-readable structured text format
- **FR-5.3**: CSV export capability for spreadsheet analysis
- **FR-5.4**: Configurable field inclusion/exclusion
- **FR-5.5**: Timestamp formatting options

### FR-6: Log File Management
**Requirement**: Manage log files efficiently
- **FR-6.1**: Automatic log rotation based on size or time
- **FR-6.2**: Configurable retention policy
- **FR-6.3**: Compression of archived logs
- **FR-6.4**: Safe concurrent write access
- **FR-6.5**: Graceful handling of disk space issues

### FR-7: Query and Analysis Tools
**Requirement**: Provide tools to analyze logged data
- **FR-7.1**: Command-line log viewer with filtering
- **FR-7.2**: Summary statistics (total calls, tokens, etc.)
- **FR-7.3**: Filter by time range, role, model, or success status
- **FR-7.4**: Token usage analysis and trending
- **FR-7.5**: Identify calls approaching token limits
- **FR-7.6**: Export filtered results

### FR-8: Integration Points
**Requirement**: Integrate seamlessly with existing Plandex architecture
- **FR-8.1**: Hook into all existing LLM client implementations
- **FR-8.2**: Preserve existing error handling behavior
- **FR-8.3**: Maintain existing configuration system compatibility
- **FR-8.4**: Support existing environment variable patterns
- **FR-8.5**: Work with existing logging infrastructure

## Non-Functional Requirements

### NFR-1: Performance
- **NFR-1.1**: Zero performance impact when logging is disabled
- **NFR-1.2**: Less than 5% performance overhead when logging is enabled
- **NFR-1.3**: Asynchronous logging to prevent blocking LLM calls
- **NFR-1.4**: Minimal memory footprint increase
- **NFR-1.5**: Efficient log file I/O operations

### NFR-2: Reliability
- **NFR-2.1**: Logging failures must not affect Plandex functionality
- **NFR-2.2**: Graceful degradation when log destination is unavailable
- **NFR-2.3**: Atomic log write operations
- **NFR-2.4**: Recovery from log file corruption
- **NFR-2.5**: Consistent logging behavior across different platforms

### NFR-3: Compatibility
- **NFR-3.1**: Zero breaking changes to existing Plandex APIs
- **NFR-3.2**: Backward compatible configuration
- **NFR-3.3**: Support for all currently supported platforms
- **NFR-3.4**: Compatible with existing build processes
- **NFR-3.5**: No new external dependencies for core functionality

### NFR-4: Security
- **NFR-4.1**: Option to exclude sensitive data from logs
- **NFR-4.2**: Secure log file permissions
- **NFR-4.3**: No exposure of API keys in logs
- **NFR-4.4**: Optional data sanitization/redaction
- **NFR-4.5**: Audit trail for log access

### NFR-5: Usability
- **NFR-5.1**: Simple enable/disable mechanism
- **NFR-5.2**: Intuitive configuration options
- **NFR-5.3**: Clear documentation and examples
- **NFR-5.4**: Helpful error messages for misconfigurations
- **NFR-5.5**: Self-documenting log format

### NFR-6: Maintainability
- **NFR-6.1**: Clean separation from core Plandex logic
- **NFR-6.2**: Modular design for easy extension
- **NFR-6.3**: Comprehensive test coverage
- **NFR-6.4**: Clear code documentation
- **NFR-6.5**: Consistent coding patterns with existing codebase

## Technical Requirements

### TR-1: Architecture Patterns
- **TR-1.1**: Use decorator/wrapper pattern for LLM client instrumentation
- **TR-1.2**: Implement interface-based design for extensibility
- **TR-1.3**: Use dependency injection where applicable
- **TR-1.4**: Maintain separation of concerns

### TR-2: Data Structures
- **TR-2.1**: Define structured log entry schema
- **TR-2.2**: Support for extensible metadata fields
- **TR-2.3**: Efficient serialization/deserialization
- **TR-2.4**: Version compatibility for log format evolution

### TR-3: Configuration Schema
```yaml
logging:
  llm_calls:
    enabled: false
    file_path: "~/.plandex/llm_calls.log"
    format: "json"  # json, text, csv
    level: "basic"  # basic, detailed
    retention_days: 7
    max_file_size: "100MB"
    include_content: true
    include_sensitive: false
    filters:
      roles: ["*"]  # or specific roles
      providers: ["*"]  # or specific providers
```

### TR-4: Environment Variables
- `PLANDEX_LLM_LOGGING`: Enable/disable logging
- `PLANDEX_LLM_LOG_FILE`: Override log file location
- `PLANDEX_LLM_LOG_LEVEL`: Set logging verbosity
- `PLANDEX_LLM_LOG_FORMAT`: Set output format
- `PLANDEX_LLM_LOG_RETENTION`: Set retention period

### TR-5: Log Entry Schema
```json
{
  "timestamp": "2025-01-15T10:30:45.123Z",
  "request_id": "req_abc123",
  "session_id": "session_xyz789",
  "role": "planner",
  "provider": "openai",
  "model": "gpt-4",
  "input": {
    "message": "...",
    "tokens": 150,
    "estimated": false
  },
  "output": {
    "response": "...",
    "tokens": 300,
    "estimated": false
  },
  "limits": {
    "context_window": 8192,
    "max_tokens": 4096
  },
  "timing": {
    "duration_ms": 1250,
    "start_time": "2025-01-15T10:30:45.123Z",
    "end_time": "2025-01-15T10:30:46.373Z"
  },
  "status": {
    "success": true,
    "error_code": null,
    "error_message": null
  },
  "metadata": {
    "temperature": 0.7,
    "max_tokens_requested": 1000
  }
}
```

## User Stories

### US-1: Developer Troubleshooting
**As a** Plandex developer experiencing token limit issues
**I want to** see detailed logs of all LLM calls with token usage
**So that** I can identify which role is consuming excessive tokens

### US-2: Performance Analysis
**As a** Plandex user concerned about API costs
**I want to** analyze token usage patterns across different tasks
**So that** I can optimize my usage and reduce costs

### US-3: Configuration Flexibility
**As a** system administrator deploying Plandex
**I want to** configure logging behavior through environment variables
**So that** I can standardize logging across different environments

### US-4: Log Analysis
**As a** developer investigating Plandex behavior
**I want to** filter and search log entries by role, time, and success status
**So that** I can quickly identify patterns and issues

### US-5: Privacy Compliance
**As a** security-conscious user
**I want to** exclude sensitive content from logs while keeping metadata
**So that** I can troubleshoot issues without exposing confidential data

## Success Criteria

### Acceptance Criteria
1. All LLM calls are captured with complete metadata
2. Token usage information enables identification of limit violations
3. Logging can be enabled/disabled without code changes
4. No performance impact when logging is disabled
5. Log analysis tools provide actionable insights
6. Integration requires no changes to existing Plandex workflows
7. Configuration is intuitive and well-documented

### Performance Benchmarks
- Logging disabled: 0% performance impact
- Logging enabled: <5% performance overhead
- Log file I/O: <10ms additional latency per call
- Memory usage increase: <50MB for typical sessions
- Log file growth: Predictable and manageable

## Deployment and Patching Strategy

### Patch Requirements
**Requirement**: The logging feature must be deployable as a non-destructive patch to a fresh Plandex repository clone

### DR-1: Repository Setup and Patching
- **DR-1.1**: Must work with a clean clone of the official Plandex repository
- **DR-1.2**: Patch must be applied programmatically without manual intervention
- **DR-1.3**: All modifications must be additive (new files) or minimal edits to existing files
- **DR-1.4**: Must preserve existing git history and allow for easy updates from upstream
- **DR-1.5**: Patch application must be idempotent (can be run multiple times safely)

### DR-2: Build and Test Integration
- **DR-2.1**: Modified Plandex must compile successfully using existing build process
- **DR-2.2**: All existing functionality must remain intact and testable
- **DR-2.3**: New logging features must be testable from command line
- **DR-2.4**: Must support standard Go build commands (`go build`, `go test`)
- **DR-2.5**: Generated binary must be functionally equivalent to original when logging disabled

### DR-3: Patch Delivery Mechanism
- **DR-3.1**: Provide automated script to clone repository and apply patch
- **DR-3.2**: Include verification steps to ensure successful patch application
- **DR-3.3**: Provide rollback mechanism to restore original state
- **DR-3.4**: Include pre-flight checks for Go version, dependencies, and environment
- **DR-3.5**: Generate patch documentation with before/after comparisons

### DR-4: Deployment Script Requirements
```bash
#!/bin/bash
# deploy-logging-patch.sh

# 1. Clone fresh Plandex repository
git clone https://github.com/plandex-ai/plandex.git plandex-with-logging
cd plandex-with-logging

# 2. Apply logging patch
# - Add new logging package files
# - Modify existing files minimally
# - Update configuration structures

# 3. Verify compilation
go mod tidy
go build ./...
go test ./...

# 4. Test basic functionality
./plandex --version
./plandex --help

# 5. Test logging functionality
export PLANDEX_LLM_LOGGING=true
./plandex new test-project
# Verify log file creation and content

# 6. Generate deployment report
```

### DR-5: File Modification Strategy
**Requirement**: Minimize changes to existing files while adding comprehensive logging

#### New Files to Add:
- `pkg/llmlog/logger.go` - Core logging infrastructure
- `pkg/llmlog/wrapper.go` - LLM client wrapper
- `pkg/llmlog/config.go` - Configuration integration
- `pkg/llmlog/formatter.go` - Output formatting
- `pkg/llmlog/analyzer.go` - Log analysis tools
- `cmd/plandex/logs.go` - CLI log viewing commands
- `test/llmlog/` - Test files for logging functionality

#### Minimal Edits to Existing Files:
- Configuration struct additions (1-2 files)
- LLM client instantiation points (2-3 files)
- Main application initialization (1 file)
- CLI command registration (1 file)

### DR-6: Patch Validation Requirements
- **DR-6.1**: Automated verification that all original tests still pass
- **DR-6.2**: Verification that logging captures expected data
- **DR-6.3**: Performance benchmarking to ensure <5% overhead
- **DR-6.4**: Configuration validation across different environments
- **DR-6.5**: Cross-platform testing (Linux, macOS, Windows if supported)

### DR-7: Documentation and Maintenance
- **DR-7.1**: Patch includes updated README with logging section
- **DR-7.2**: Configuration examples and troubleshooting guide
- **DR-7.3**: Clear instructions for updating patch when upstream changes
- **DR-7.4**: Documentation of all modified files and rationale
- **DR-7.5**: Migration path for users of original Plandex

## Dependencies

### Internal Dependencies
- Existing Plandex LLM client interfaces
- Current configuration management system
- Existing logging infrastructure
- Build and deployment processes

### External Dependencies
- Go standard library (no new external dependencies for core functionality)
- Optional: tiktoken library for accurate OpenAI token counting
- File system access for log storage
- JSON marshaling/unmarshaling capabilities

### Deployment Dependencies
- Git for repository cloning
- Go compiler (same version as required by Plandex)
- Bash or equivalent shell for deployment script
- Write access to deployment directory

## Constraints

### Technical Constraints
- Must maintain Go version compatibility with existing Plandex
- Cannot modify existing public API interfaces
- Must work across all supported platforms (Windows, macOS, Linux)
- Cannot introduce breaking changes to configuration

### Business Constraints
- Zero disruption to existing users
- Minimal development time investment
- No additional infrastructure requirements
- Must be maintainable by existing team

## Risks and Mitigations

### Risk 1: Performance Impact
**Mitigation**: Asynchronous logging, benchmarking, feature flags

### Risk 2: Log File Size Growth
**Mitigation**: Automatic rotation, compression, retention policies

### Risk 3: Integration Complexity
**Mitigation**: Wrapper pattern, comprehensive testing, gradual rollout

### Risk 4: Configuration Complexity
**Mitigation**: Sensible defaults, clear documentation, validation

### Risk 5: Privacy Concerns
**Mitigation**: Content filtering options, redaction capabilities, clear policies