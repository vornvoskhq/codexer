# Codex-Lite Integration Roadmap

## Overview
This document outlines the plan to integrate the codex-lite agent with the Phase 2 tools, enabling autonomous tool use for file operations, git, code analysis, scaffolding, and refactoring.

## Current Architecture

### Codex-Lite Agent
- Simple chat interface to LLM (OpenRouter)
- Basic command parsing (patch, shell)
- No direct tool integration

### Phase 2 Tools (in `/integration`)
- File operations (`file_tools.py`)
- Git operations (`git_tools.py`)
- Code analysis (`code_analysis.py`)
- Project tools (`project_tools.py`)
- Advanced agent tools (`advanced_agent_tools.py`)
- IDE tools (`ide_tools.py`)
- Specialized tools (`specialized_tools.py`)

## Integration Plan

### 1. Tool Registration System
```python
class ToolRegistry:
    def __init__(self):
        self.tools = {}
    
    def register(self, name, func, description, schema):
        self.tools[name] = {
            'function': func,
            'description': description,
            'schema': schema
        }
```

### 2. Tool Descriptions for LLM
Generate JSON schema for each tool to help the LLM understand:
- Tool name and purpose
- Required parameters
- Expected output format
- Example usage

### 3. Agent Loop Enhancement
```python
class EnhancedCodexAgent:
    def __init__(self, tool_registry):
        self.tool_registry = tool_registry
        self.llm = OpenRouterClient()
    
    def process_input(self, user_input):
        # 1. Use LLM to determine if tool use is needed
        # 2. If tool is needed, extract parameters
        # 3. Execute tool with parameters
        # 4. Format and return results
        pass
```

### 4. Tool Execution Flow
1. User sends request to agent
2. Agent determines if tool use is needed
3. If yes:
   - Select appropriate tool
   - Extract parameters from user input
   - Execute tool with parameters
   - Format and return results
4. If no:
   - Generate response using LLM directly

## Implementation Phases

### Phase 1: Basic Tool Integration (Week 1-2)
- [ ] Set up tool registration system
- [ ] Implement basic file operations (read, write, list)
- [ ] Add git status and commit capabilities
- [ ] Create simple agent loop with tool use

### Phase 2: Advanced Tool Integration (Week 3-4)
- [ ] Add code analysis tools
- [ ] Implement project scaffolding
- [ ] Add refactoring tools
- [ ] Enhance error handling and validation

### Phase 3: Optimization & UX (Week 5-6)
- [ ] Improve tool selection accuracy
- [ ] Add tool chaining capabilities
- [ ] Implement conversation history
- [ ] Add user feedback mechanisms

## Tool Examples

### File Operations
```python
{
    "name": "read_file",
    "description": "Read the contents of a file",
    "parameters": {
        "path": {"type": "string", "description": "Path to the file"}
    },
    "returns": "File contents as string"
}
```

### Git Operations
```python
{
    "name": "git_status",
    "description": "Get the git status of the repository",
    "parameters": {
        "path": {"type": "string", "description": "Path to git repository"}
    },
    "returns": "Git status output"
}
```

## Testing Implementation

### Test Directory Structure
```
tests/
├── unit/
│   ├── test_file_operations.py
│   ├── test_git_operations.py
│   └── test_code_analysis.py
├── integration/
│   ├── test_workflows.py
│   └── test_tool_chaining.py
├── e2e/
│   └── test_full_workflows.py
└── conftest.py  # Shared fixtures
```

### Example Test Implementations

#### 1. Unit Test Example (`test_file_operations.py`)
```python
import pytest
from integration.tools.file_tools import read_file, write_file
import os

def test_read_file(tmp_path):
    # Setup
    test_file = tmp_path / "test.txt"
    test_content = "Hello, World!"
    test_file.write_text(test_content)
    
    # Test
    content = read_file(str(test_file))
    assert content == test_content

def test_read_nonexistent_file():
    with pytest.raises(FileNotFoundError):
        read_file("nonexistent.txt")
```

#### 2. Integration Test Example (`test_workflows.py`)
```python
def test_git_workflow(tmp_path, git_repo):
    # Create test file
    test_file = tmp_path / "test.py"
    test_file.write_text("def hello(): return 'world'")
    
    # Stage file
    git_repo.git.add(str(test_file))
    assert "test.py" in git_repo.git.status()
    
    # Commit
    git_repo.git.commit("-m", "Add test file")
    assert "test.py" in git_repo.git.log()
```

#### 3. End-to-End Test Example (`test_full_workflows.py`)
```python
def test_project_initialization(tmp_path, mock_llm):
    # Initialize project
    result = run_codex_command("create project my_project")
    
    # Verify project structure
    assert (tmp_path / "my_project").exists()
    assert (tmp_path / "my_project" / "README.md").exists()
    
    # Verify git repository
    assert (tmp_path / "my_project" / ".git").is_dir()
```

### Test Fixtures (`conftest.py`)
```python
import pytest
import git
import tempfile
from pathlib import Path

@pytest.fixture
def git_repo(tmp_path):
    # Create a git repo for testing
    repo = git.Repo.init(tmp_path)
    # Set up test user
    repo.config_writer().set_value("user", "name", "Test User").release()
    repo.config_writer().set_value("user", "email", "test@example.com").release()
    return repo

@pytest.fixture
def mock_llm(monkeypatch):
    # Mock LLM responses for testing
    def mock_query(*args, **kwargs):
        return "Mocked LLM response"
    
    monkeypatch.setattr("codex.llm.openrouter.query", mock_query)
```

## Test Automation

### CI/CD Pipeline
```yaml
# .github/workflows/test.yml
name: Test Suite

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Python
      uses: actions/setup-python@v4
      with:
        python-version: '3.10'
    
    - name: Install dependencies
      run: |
        python -m pip install --upgrade pip
        pip install -r requirements-dev.txt
    
    - name: Run tests
      run: |
        pytest tests/ --cov=integration --cov-report=xml
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
```

## Testing Strategy

### 1. Unit Tests
- **Tool Functionality**: Test each tool in isolation
  - File operations (read, write, list)
  - Git operations (status, commit, branch)
  - Code analysis (linting, type checking)
  - Project scaffolding (template generation)
  
```python
def test_read_file():
    # Test file reading with valid path
    # Test file reading with invalid path
    # Test file reading with no permissions
    pass
```

### 2. Integration Tests
- **Tool Chaining**: Test multiple tools working together
  - File write → Git add → Git commit
  - Code analysis → Auto-fix suggestions
  - Project scaffolding → Dependency installation

```python
def test_git_workflow():
    # Create test file
    # Stage file with git
    # Commit changes
    # Verify commit history
    pass
```

### 3. End-to-End Tests
- **Complete Workflows**:
  - Full project initialization
  - Code modification and version control
  - Refactoring operations
  - Dependency management

### 4. Mock Testing
- **External Services**:
  - Mock LLM responses
  - Simulate git operations
  - Test error conditions

### 5. Performance Testing
- **Benchmarking**:
  - Tool execution time
  - Memory usage
  - Concurrent operations

### 6. User Acceptance Testing (UAT)
- **Real-world Scenarios**:
  - Developer workflows
  - Common use cases
  - Edge cases and error handling

## Future Enhancements
1. Tool versioning
2. Access control for tools
3. Tool execution history
4. Performance monitoring
5. Plugin system for custom tools

## Dependencies
- Python 3.8+
- `gitpython` for git operations
- `pydantic` for schema validation
- `pytest` for testing

## Getting Started
1. Install dependencies: `pip install -r requirements.txt`
2. Set up environment variables
3. Run tests: `pytest tests/`
4. Start the agent: `python -m codex.cli.main`
