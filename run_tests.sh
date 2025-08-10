#!/usr/bin/env bash
# A robust and extensible test runner for Python projects

# Exit immediately if a command exits with a non-zero status
set -euo pipefail

# ============================================
# Global Variables
# ============================================

# Set the project root directory
readonly PROJECT_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
cd "$PROJECT_ROOT"

# Simple color handling - disable colors by default for now to avoid issues
readonly COLOR_RED=''
readonly COLOR_GREEN=''
readonly COLOR_YELLOW=''
readonly COLOR_BLUE=''
readonly COLOR_MAGENTA=''
readonly COLOR_CYAN=''
readonly COLOR_RESET=''
readonly BOLD=''

# ============================================
# Helper Functions
# ============================================

# Logging functions
log() {
    local level="$1"
    local message="${*:2}"
    local timestamp
    timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    
    # Skip if verbosity is too low
    [ "${VERBOSITY:-1}" -eq 0 ] && [ "$level" != "ERROR" ] && return 0
    [ "${VERBOSITY:-1}" -eq 1 ] && [ "$level" = "DEBUG" ] && return 0
    
    case "$level" in
        "DEBUG")
            echo -e "${COLOR_CYAN}[$timestamp] [DEBUG] $message${COLOR_RESET}" >&2
            ;;
        "INFO")
            echo -e "${COLOR_BLUE}[$timestamp] [INFO] $message${COLOR_RESET}"
            ;;
        "SUCCESS")
            echo -e "${COLOR_GREEN}[$timestamp] [SUCCESS] $message${COLOR_RESET}"
            ;;
        "WARNING")
            echo -e "${COLOR_YELLOW}[$timestamp] [WARNING] $message${COLOR_RESET}" >&2
            ;;
        "ERROR")
            echo -e "${COLOR_RED}[$timestamp] [ERROR] $message${COLOR_RESET}" >&2
            ;;
        *)
            echo -e "[$timestamp] [$level] $message"
            ;;
    esac
}

# Print error message and exit
die() {
    log "ERROR" "$1"
    if [ "${VERBOSITY:-1}" -ge 3 ]; then
        log "DEBUG" "Stack trace:"
        local i=0
        while caller $i; do
            i=$((i+1))
        done
    fi
    exit 1
}

# Print warning message
warn() {
    log "WARNING" "$1"
}

# Print success message
success() {
    log "SUCCESS" "$1"
}

# Print info message
info() {
    log "INFO" "$1"
}

# Print debug message
debug() {
    if [ "${VERBOSITY:-1}" -ge 3 ]; then
        log "DEBUG" "$1"
    fi
}

# Print section header
section() {
    echo -e "\n${BOLD}${COLOR_MAGENTA}=== $1 ===${COLOR_RESET}"
}

# ============================================
# Configuration
# ============================================

# Default configuration - can be overridden in .testconfig
readonly DEFAULT_CONFIG='# Test configuration file - modify these values as needed

# Python command to use (python, python3, etc.)
PYTHON_CMD="python3"

# Virtual environment directory
VENV_DIR=".venv"

# Test directories to search for tests (space-separated)
TEST_DIRS="tests/"

# Test patterns to include (space-separated)
TEST_PATTERNS="test_*.py"

# Test patterns to exclude (space-separated)
TEST_EXCLUDE_PATTERNS=""

# Requirements files (space-separated, in order of installation)
REQUIREMENTS_FILES="requirements-dev.txt requirements-test.txt requirements.txt"

# Additional test commands to run (space-separated)
# Format: "command1:description1,command2:description2"
CUSTOM_TEST_COMMANDS=""

# Project-specific setup command (e.g., "npm install", "bundle install")
SETUP_COMMAND=""

# Whether to run linters (1 = yes, 0 = no)
RUN_LINTERS=1

# Linter commands (space-separated)
LINTER_COMMANDS="flake8"

# Verbosity level (0 = quiet, 1 = normal, 2 = verbose, 3 = debug)
VERBOSITY=1

# Color output (1 = enable, 0 = disable)
COLOR_OUTPUT=1

# Maximum number of test failures before aborting (0 = unlimited)
MAX_FAILURES=0

# Timeout for individual tests in seconds (0 = no timeout)
TEST_TIMEOUT=60

# Whether to run tests in parallel (1 = yes, 0 = no)
PARALLEL_TESTS=0

# Number of parallel jobs (only used if PARALLEL_TESTS=1)
NUM_JOBS=4

# Whether to generate coverage report (1 = yes, 0 = no)
GENERATE_COVERAGE=0

# Coverage report format (html, xml, report)
COVERAGE_FORMAT="report"

# Whether to open coverage report in browser (only if GENERATE_COVERAGE=1 and COVERAGE_FORMAT=html)
OPEN_COVERAGE_REPORT=0'

# Logging functions
log() {
    local level="$1"
    local message="${*:2}"
    local timestamp
    timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    
    # Skip if verbosity is too low
    [ "${VERBOSITY:-1}" -eq 0 ] && [ "$level" != "ERROR" ] && return 0
    [ "${VERBOSITY:-1}" -eq 1 ] && [ "$level" = "DEBUG" ] && return 0
    
    case "$level" in
        "DEBUG")
            echo -e "${COLOR_CYAN}[$timestamp] [DEBUG] $message${COLOR_RESET}" >&2
            ;;
        "INFO")
            echo -e "${COLOR_BLUE}[$timestamp] [INFO] $message${COLOR_RESET}"
            ;;
        "SUCCESS")
            echo -e "${COLOR_GREEN}[$timestamp] [SUCCESS] $message${COLOR_RESET}"
            ;;
        "WARNING")
            echo -e "${COLOR_YELLOW}[$timestamp] [WARNING] $message${COLOR_RESET}" >&2
            ;;
        "ERROR")
            echo -e "${COLOR_RED}[$timestamp] [ERROR] $message${COLOR_RESET}" >&2
            ;;
        *)
            echo -e "[$timestamp] [$level] $message"
            ;;
    esac
}

# Print error message and exit
die() {
    log "ERROR" "$1"
    if [ "${VERBOSITY:-1}" -ge 3 ]; then
        log "DEBUG" "Stack trace:"
        local i=0
        while caller $i; do
            i=$((i+1))
        done
    fi
    exit 1
}

# Print warning message
warn() {
    log "WARNING" "$1"
}

# Print success message
success() {
    log "SUCCESS" "$1"
}

# Print info message
info() {
    log "INFO" "$1"
}

# Print debug message
debug() {
    if [ "${VERBOSITY:-1}" -ge 3 ]; then
        log "DEBUG" "$1"
    fi
}

# Print section header
section() {
    echo -e "\n${BOLD}${COLOR_MAGENTA}=== $1 ===${COLOR_RESET}"
}

# Check if a command exists
command_exists() {
    if ! command -v "$1" >/dev/null 2>&1; then
        debug "Command not found: $1"
        return 1
    fi
    return 0
}

# Cross-platform version of realpath
realpath() {
    if command_exists realpath; then
        command realpath "$@"
    elif command_exists python3; then
        python3 -c "import os, sys; print(os.path.realpath(sys.argv[1]))" "$1"
    elif command_exists python; then
        python -c "import os, sys; print(os.path.realpath(sys.argv[1]))" "$1"
    else
        # Fallback for systems without realpath or Python
        local path="$1"
        if [ -d "$path" ]; then
            (cd "$path" && pwd -P)
        else
            local dir
            local file
            dir=$(cd "$(dirname "$path")" && pwd -P)
            file=$(basename "$path")
            echo "$dir/$file"
        fi
    fi
}

# Check if a file exists and is readable
file_exists() {
    [ -r "$1" ] && [ -f "$1" ]
}

# Check if a directory exists and is accessible
dir_exists() {
    [ -r "$1" ] && [ -d "$1" ]
}

# Run a command with error handling
run_command() {
    local cmd=("$@")
    local exit_code=0
    
    debug "Running command: ${cmd[*]}"
    
    if [ "$VERBOSITY" -ge 2 ]; then
        "${cmd[@]}" 
        exit_code=$?
    else
        "${cmd[@]}" >/dev/null 2>&1
        exit_code=$?
    fi
    
    if [ $exit_code -ne 0 ]; then
        debug "Command failed with exit code $exit_code: ${cmd[*]}"
    fi
    
    return $exit_code
}

# Load configuration from .testconfig or create it if it doesn't exist
load_config() {
    local config_file="${1:-.testconfig}"
    
    if [ ! -f "$config_file" ]; then
        info "Creating default configuration file: $config_file"
        echo "$DEFAULT_CONFIG" > "$config_file"
        # shellcheck source=/dev/null
        . "$config_file"
        return 0
    fi
    
    debug "Loading configuration from: $config_file"
    
    # Check if the config file is readable
    if [ ! -r "$config_file" ]; then
        warn "Configuration file $config_file exists but is not readable. Using defaults."
        return 1
    fi
    
    # Check for dangerous patterns in the config file
    if grep -q -E '\$\{?[a-zA-Z_][a-zA-Z0-9_]*\([^}]*\)' "$config_file"; then
        warn "Potentially dangerous shell code detected in $config_file. Aborting for security."
        return 1
    fi
    
    # Load the configuration safely
    # shellcheck source=/dev/null
    if ! . "$config_file"; then
        warn "Failed to load configuration from $config_file. Using defaults."
        return 1
    fi
}

# Show help message
show_help() {
    cat << EOF
Usage: $0 [options] [pytest-options]

A robust and extensible test runner for Python projects.

Options:
  -h, --help       Show this help message and exit
  --no-venv        Skip virtual environment creation/activation
  --install        Install dependencies before running tests
  --uv             Use uv for dependency management (faster)
  --ci             Run in CI mode (non-interactive, with colors)
  --setup          Run project setup commands
  --lint           Run linters
  --all            Run all test types (unit, integration, lint, etc.)
  --coverage       Generate test coverage report
  --codex-lite     Run codex-lite specific tests (if applicable)

Any additional arguments are passed to pytest.

Configuration:
  This script is configured via the .testconfig file in your project root.
  A default configuration will be created if it doesn't exist.

Examples:
  $0                    # Run all tests
  $0 tests/unit/        # Run tests in a specific directory
  $0 -k test_feature    # Run tests matching a specific pattern
  $0 --coverage         # Generate coverage report
  $0 --all              # Run all tests, linters, and checks

EOF
    exit 0
}

# Initialize the test environment
init_environment() {
    if [[ "$NO_VENV" -eq 0 ]]; then
        setup_virtualenv
    fi
    
    if [[ "$INSTALL_DEPS" -eq 1 ]]; then
        install_dependencies
    fi
    
    if [[ "$RUN_SETUP" -eq 1 && -n "$SETUP_COMMAND" ]]; then
        run_setup
    fi
}

# Set up Python virtual environment
setup_virtualenv() {
    section "Setting up Python Virtual Environment"
    
    # Define the virtual environment activation script path
    local VENV_ACTIVATE="${VENV_DIR}/bin/activate"
    if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" || "$OSTYPE" == "win32" ]]; then
        VENV_ACTIVATE="${VENV_DIR}/Scripts/activate"
    fi
    
    # Check if virtual environment already exists
    if [ -d "$VENV_DIR" ]; then
        if [ -f "$VENV_ACTIVATE" ]; then
            info "Using existing virtual environment at $VENV_DIR"
            # shellcheck source=/dev/null
            if ! source "$VENV_ACTIVATE" 2>/dev/null; then
                warn "Failed to activate virtual environment. Recreating..."
                rm -rf "$VENV_DIR"
            else
                # Verify the virtual environment is working
                if ! command -v python >/dev/null 2>&1; then
                    warn "Virtual environment appears to be corrupted. Recreating..."
                    rm -rf "$VENV_DIR"
                else
                    return 0  # Virtual environment is good
                fi
            fi
        else
            warn "Virtual environment directory exists but is incomplete. Recreating..."
            rm -rf "$VENV_DIR"
        fi
    fi
    
    # Create new virtual environment
    info "Creating new virtual environment in $VENV_DIR..."
    if ! "$PYTHON_CMD" -m venv "$VENV_DIR"; then
        die "Failed to create virtual environment. Please check your Python installation and try again."
    fi
    
    # Activate the new virtual environment
    # shellcheck source=/dev/null
    if ! source "$VENV_ACTIVATE"; then
        die "Failed to activate virtual environment. Please check the path: $VENV_ACTIVATE"
    fi
    
    # Upgrade pip and setuptools in the new environment
    info "Upgrading pip and setuptools..."
    if ! python -m pip install --upgrade pip setuptools wheel; then
        warn "Failed to upgrade pip and setuptools. Some dependencies might fail to install."
    fi
    
    success "Virtual environment setup completed successfully"
    return 0
}

# Install project dependencies
install_dependencies() {
    section "Installing Dependencies"
    
    # Ensure we're in a virtual environment
    if [ -z "${VIRTUAL_ENV:-}" ]; then
        warn "Not in a virtual environment. Activating..."
        # shellcheck source=/dev/null
        if ! source "$VENV_ACTIVATE"; then
            die "Failed to activate virtual environment for dependency installation"
        fi
    fi
    
    # Create a temporary requirements file to consolidate all dependencies
    local temp_req_file
    temp_req_file=$(mktemp)
    
    # Function to add requirements from a file if it exists
    add_requirements() {
        local req_file="$1"
        if [ -f "$req_file" ]; then
            echo "# From $req_file" >> "$temp_req_file"
            cat "$req_file" >> "$temp_req_file"
            echo "" >> "$temp_req_file"
            info "Found requirements file: $req_file"
            return 0
        fi
        return 1
    }
    
    # Process each requirements file
    local req_file
    local found_any=0
    for req_file in $REQUIREMENTS_FILES; do
        if add_requirements "$req_file"; then
            found_any=1
        fi
    done
    
    if [ "$found_any" -eq 0 ]; then
        warn "No requirements files found. Looking for common filenames..."
        for req_file in requirements.txt requirements-dev.txt requirements-test.txt; do
            if add_requirements "$req_file"; then
                found_any=1
            fi
        done
    fi
    
    if [ "$found_any" -eq 0 ]; then
        warn "No requirements files found. Installing only the package in development mode."
    else
        info "Checking for missing dependencies..."
        
        # Function to check if all requirements are already installed
        check_requirements() {
            local req_file="$1"
            local missing=0
            
            # Check each line in requirements file
            while IFS= read -r line || [ -n "$line" ]; do
                # Skip comments and empty lines
                [[ "$line" =~ ^[[:space:]]*# ]] && continue
                [[ -z "${line// }" ]] && continue
                
                # Extract package name (handling different requirement formats)
                local pkg_name="${line%%[<=>!~]*}"
                pkg_name="${pkg_name//[<>=!~]}"
                pkg_name="${pkg_name//[[:space:]]}"
                
                # Skip empty package names
                [ -z "$pkg_name" ] && continue
                
                # Check if package is installed
                if ! python -c "import ${pkg_name%%[.[*#%^&]}" &>/dev/null; then
                    debug "Package not found: $pkg_name"
                    missing=1
                    break
                fi
            done < "$req_file"
            
            return $missing
        }
        
        # Check if all requirements are already installed
        if check_requirements "$temp_req_file"; then
            info "All dependencies are already installed."
        else
            info "Installing missing dependencies..."
            # First, try to install with --upgrade-strategy=only-if-needed to avoid reinstalling
            if ! python -m pip install --upgrade-strategy=only-if-needed -r "$temp_req_file"; then
                warn "Failed to install some dependencies with --upgrade-strategy. Attempting standard install..."
                # If that fails, try a standard install but with --no-deps to avoid reinstalling dependencies
                if ! python -m pip install --no-deps -r "$temp_req_file"; then
                    warn "Failed to install some dependencies. Attempting to continue..."
                fi
            fi
        fi
    fi
    
    # Install the package in development mode if setup.py exists
    if [ -f "setup.py" ]; then
        info "Installing package in development mode..."
        if ! python -m pip install -e .; then
            warn "Failed to install package in development mode"
        fi
    fi
    
    # Clean up
    rm -f "$temp_req_file"
    
    success "Dependency installation completed"
    return 0
    local req_installed=0
    
    for req_file in $REQUIREMENTS_FILES; do
        if [ -f "$req_file" ]; then
            info "Installing from $req_file..."
            if [[ "$USE_UV" -eq 1 ]] && command_exists uv; then
                uv pip install -r "$req_file"
            else
                pip install -r "$req_file"
            fi
            req_installed=1
        fi
    done
    
    # Install test dependencies if no requirements files were found
    if [[ "$req_installed" -eq 0 ]]; then
        warn "No requirements files found, installing common test dependencies..."
        local test_deps=(
            "pytest"
            "pytest-cov"
            "pytest-asyncio"
            "pytest-mock"
            "pytest-playwright"
            "playwright"
            "requests"
        )
        
        if [[ "$USE_UV" -eq 1 ]] && command_exists uv; then
            uv pip install "${test_deps[@]}"
        else
            pip install "${test_deps[@]}"
        fi
        
        # Install Playwright browsers
        if command_exists playwright; then
            info "Installing Playwright browsers..."
            playwright install
        fi
    fi
}

# Run project setup commands
run_setup() {
    if [ -z "$SETUP_COMMAND" ]; then
        debug "No setup command specified"
        return 0
    fi
    
    section "Running Project Setup"
    
    # Check if we're in a virtual environment
    if [ -z "${VIRTUAL_ENV:-}" ]; then
        warn "Not in a virtual environment. Setup might fail."
    fi
    
    # Split the setup command into an array
    IFS=' ' read -r -a setup_cmd <<< "$SETUP_COMMAND"
    
    # Handle common setup commands
    case "${setup_cmd[0]}" in
        npm|yarn|pnpm)
            if ! command_exists "${setup_cmd[0]}"; then
                warn "${setup_cmd[0]} is not installed. Skipping setup."
                return 1
            fi
            ;;
        python|python3|pip)
            # Use the Python from the virtual environment if available
            if [ -n "${VIRTUAL_ENV:-}" ]; then
                setup_cmd[0]="python"
            fi
            ;;
    esac
    
    info "Running setup command: ${setup_cmd[*]}"
    
    # Execute the setup command
    if ! "${setup_cmd[@]}"; then
        warn "Setup command failed with status $?"
        return 1
    fi
    
    success "Project setup completed successfully"
    return 0
}

# Run linters
run_linters() {
    if [ "$RUN_LINTERS" -eq 0 ]; then
        info "Skipping linters as requested"
        return 0
    fi
    
    section "Running Linters"
    local lint_passed=1
    local any_linter_ran=0
    
    # Check if there are any Python files to lint
    local python_files
    python_files=$(find . -name "*.py" -not -path "*$VENV_DIR/*" -not -path "*/.git/*" | wc -l)
    
    if [ "$python_files" -eq 0 ]; then
        info "No Python files found to lint"
        return 0
    fi
    
    for linter in $LINTER_COMMANDS; do
        if ! command_exists "$linter"; then
            warn "Linter '$linter' not found. Skipping..."
            continue
        fi
        
        info "Running $linter..."
        any_linter_ran=1
        
        case "$linter" in
            flake8)
                local flake8_args=()
                [ -f ".flake8" ] && flake8_args+=("--config" ".flake8")
                flake8 "${flake8_args[@]}" .
                lint_passed=$((lint_passed & $?))
                ;;
            pylint)
                local pylint_args=()
                [ -f "pylintrc" ] && pylint_args+=("--rcfile=pylintrc")
                [ -f "setup.cfg" ] && pylint_args+=("--rcfile=setup.cfg")
                pylint "${pylint_args[@]}" .
                lint_passed=$((lint_passed & $?))
                ;;
            mypy)
                if [ -f "mypy.ini" ] || [ -f ".mypy.ini" ] || [ -f "setup.cfg" ]; then
                    mypy .
                else
                    mypy --ignore-missing-imports .
                fi
                lint_passed=$((lint_passed & $?))
                ;;
            *)
                # Generic linter execution
                if [ -x "$(command -v "$linter")" ]; then
                    "$linter"
                    lint_passed=$((lint_passed & $?))
                else
                    warn "Linter '$linter' is not executable. Skipping..."
                fi
                ;;
        esac
    done
    
    if [ "$any_linter_ran" -eq 0 ]; then
        warn "No linters were executed. Make sure the linters are installed and in your PATH."
        return 1
    fi
    
    if [ $lint_passed -eq 0 ]; then
        success "All linters passed"
    else
        warn "Some linters reported issues"
    fi
    
    return $lint_passed
}

# Run custom test commands
run_custom_tests() {
    if [ -z "$CUSTOM_TEST_COMMANDS" ]; then
        return 0
    fi
    
    section "Running custom test commands"
    local test_failed=0
    
    # Split commands by comma and process each
    IFS=',' read -ra commands <<< "$CUSTOM_TEST_COMMANDS"
    for cmd_pair in "${commands[@]}"; do
        # Split command and description
        IFS=':' read -r cmd desc <<< "$cmd_pair"
        if [ -n "$cmd" ]; then
            info "Running: ${desc:-$cmd}"
            if ! eval "$cmd"; then
                warn "Command failed: $cmd"
                test_failed=1
            fi
        fi
    done
    
    return $test_failed
}

# Run pytest tests
run_pytest() {
    section "Running Tests with Pytest"
    
    # Check if pytest is installed
    if ! command_exists pytest; then
        warn "pytest not found. Installing with required plugins..."
        if ! python -m pip install pytest pytest-cov pytest-timeout; then
            die "Failed to install pytest and required plugins"
        fi
    else
        # Ensure required plugins are installed
        if ! python -c "import pytest_cov" 2>/dev/null; then
            info "Installing pytest-cov plugin..."
            python -m pip install pytest-cov
        fi
        if ! python -c "import pytest_timeout" 2>/dev/null; then
            info "Installing pytest-timeout plugin..."
            python -m pip install pytest-timeout
        fi
    fi
    
    # Build pytest arguments
    local pytest_args=(
        "-v"           # Verbose output
        "--tb=short"   # Shorter tracebacks
    )
    
    # Add coverage if requested
    if [ "$GENERATE_COVERAGE" -eq 1 ]; then
        pytest_args+=(
            "--cov=."
            "--cov-report=$COVERAGE_FORMAT"
        )
        
        if [ "$COVERAGE_FORMAT" = "html" ] && [ "$OPEN_COVERAGE_REPORT" -eq 1 ]; then
            pytest_args+=("--cov-report=html:htmlcov")
        fi
    fi
    
    # Add maxfail if specified
    if [ "$MAX_FAILURES" -gt 0 ]; then
        pytest_args+=("--maxfail=$MAX_FAILURES")
    fi
    
    # Add timeout if specified
    if [ "$TEST_TIMEOUT" -gt 0 ]; then
        pytest_args+=("--timeout=$TEST_TIMEOUT")
    fi
    
    # Add test patterns if specified
    if [ -n "$TEST_PATTERNS" ]; then
        # Convert space-separated patterns to array
        IFS=' ' read -r -a patterns <<< "$TEST_PATTERNS"
        for pattern in "${patterns[@]}"; do
            pytest_args+=("$pattern")
        done
    fi
    
    # Add test exclude patterns if specified
    if [ -n "$TEST_EXCLUDE_PATTERNS" ]; then
        # Convert space-separated patterns to array
        IFS=' ' read -r -a exclude_patterns <<< "$TEST_EXCLUDE_PATTERNS"
        for pattern in "${exclude_patterns[@]}"; do
            pytest_args+=("--ignore=$pattern")
        done
    fi
    
    # Add test paths if they exist
    local test_paths=()
    local found_tests=0
    
    # Check if specific test directories are provided
    if [ -n "$TEST_DIRS" ]; then
        for test_dir in $TEST_DIRS; do
            if [ -d "$test_dir" ]; then
                test_paths+=("$test_dir")
                found_tests=1
                debug "Added test directory: $test_dir"
            else
                warn "Test directory not found: $test_dir"
            fi
        done
    fi
    
    # If no test directories were found, look for common test locations
    if [ $found_tests -eq 0 ]; then
        warn "No valid test directories found in TEST_DIRS. Searching for common test locations..."
        
        local common_test_dirs=("tests" "test" "src/tests" "src/test" "unit_tests" "integration_tests")
        
        for test_dir in "${common_test_dirs[@]}"; do
            if [ -d "$test_dir" ]; then
                test_paths+=("$test_dir")
                found_tests=1
                info "Found tests in: $test_dir"
                break
            fi
        done
        
        # If still no tests found, use the current directory
        if [ $found_tests -eq 0 ]; then
            warn "No test directories found. Running tests in the current directory..."
            test_paths=(".")
        fi
    fi
    
    # Run pytest with the specified arguments
    info "Running tests in: ${test_paths[*]}"
    debug "pytest command: python -m pytest ${pytest_args[*]} ${test_paths[*]}"
    
    local start_time
    start_time=$(date +%s)
    
    # Run pytest and capture the exit code
    python -m pytest "${pytest_args[@]}" "${test_paths[@]}"
    local pytest_exit_code=$?
    
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Check the pytest exit code
    if [ $pytest_exit_code -eq 0 ]; then
        success "All tests passed (took ${duration}s)"
        return 0
    elif [ $pytest_exit_code -eq 1 ]; then
        # Pytest returns 1 when tests failed
        warn "Some tests failed (took ${duration}s)"
    elif [ $pytest_exit_code -eq 2 ]; then
        # Pytest returns 2 when interrupted by user
        warn "Test run was interrupted (took ${duration}s)"
    elif [ $pytest_exit_code -eq 3 ]; then
        # Pytest returns 3 for internal error
        warn "Internal error occurred during test execution (took ${duration}s)"
    elif [ $pytest_exit_code -eq 4 ]; then
        # Pytest returns 4 for command line usage error
        warn "Command line usage error (took ${duration}s)"
    elif [ $pytest_exit_code -eq 5 ]; then
        # Pytest returns 5 when no tests were collected
        warn "No tests were collected (took ${duration}s)"
    else
        warn "Tests completed with unknown status (exit code: $pytest_exit_code, took ${duration}s)"
    fi
    
    # If running in CI, output test results summary
    if [ -n "${CI:-}" ]; then
        info "Generating test results summary..."
        python -m pytest "${test_paths[@]}" --junitxml="test-results.xml"
    fi
    
    return $pytest_exit_code
    
    # Open coverage report if requested
    if [ "$GENERATE_COVERAGE" -eq 1 ] && [ "$COVERAGE_FORMAT" = "html" ] && [ "$OPEN_COVERAGE_REPORT" -eq 1 ]; then
        if [ -f "htmlcov/index.html" ]; then
            info "Opening coverage report in default browser..."
            if command -v xdg-open >/dev/null; then
                xdg-open "htmlcov/index.html"
            elif command -v open >/dev/null; then
                open "htmlcov/index.html"
            elif command -v start >/dev/null; then
                start "htmlcov/index.html"
            fi
        fi
    fi
    
    return 0
}

# Run codex-lite specific tests (if applicable)
run_codex_lite_tests() {
    # Only run if codex-lite directory exists and --codex-lite flag is set
    if [[ "$RUN_CODEX_LITE" -eq 0 || ! -d "codex-lite" ]]; then
        return 0
    fi
    
    section "Running codex-lite CLI Tests"
    
    local original_dir
    original_dir=$(pwd)
    
    # Change to codex-lite directory
    cd codex-lite || {
        warn "Failed to change to codex-lite directory"
        return 1
    }
    
    # Check if codex command is available
    if ! command -v codex &> /dev/null; then
        warn "'codex' command not found in PATH. Some tests may fail."
    fi

    # Run basic CLI tests
    local tests_passed=0
    local total_tests=0

    # Test 1: Check codex --version
    ((total_tests++))
    echo -n "Test $total_tests: codex --version ... "
    if codex --version &> /dev/null; then
        echo "PASSED"
        ((tests_passed++))
    else
        echo "FAILED"
    fi

    # Test 2: Check codex --help
    ((total_tests++))
    echo -n "Test $total_tests: codex --help ... "
    if codex --help &> /dev/null; then
        echo "PASSED"
        ((tests_passed++))
    else
        echo "FAILED"
    fi

    # Add more codex-lite specific tests here

    # Return to original directory
    cd "$original_dir" || return 1

    # Print test summary
    echo -e "\ncodex-lite CLI tests: $tests_passed/$total_tests passed"
    
    # Return non-zero if any tests failed
    [ $tests_passed -eq $total_tests ]
}

# Main function
main() {
    # Initialize variables
    local INSTALL_DEPS=0
    local USE_UV=0
    local NO_VENV=0
    local RUN_CODEX_LITE=0
    local RUN_COVERAGE=0
    local RUN_LINTERS=0
    local RUN_SETUP=0
    local PYTEST_ARGS=()
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                ;;
            --no-venv)
                NO_VENV=1
                shift
                ;;
            --install)
                INSTALL_DEPS=1
                shift
                ;;
            --uv)
                USE_UV=1
                shift
                ;;
            --ci)
                # Set CI mode (non-interactive, with colors)
                export CI=1
                shift
                ;;
            --setup)
                RUN_SETUP=1
                shift
                ;;
            --lint|--linter|--linters)
                RUN_LINTERS=1
                shift
                ;;
            --all)
                INSTALL_DEPS=1
                RUN_LINTERS=1
                RUN_SETUP=1
                shift
                ;;
            --coverage)
                RUN_COVERAGE=1
                shift
                ;;
            --codex-lite)
                RUN_CODEX_LITE=1
                shift
                ;;
            --)
                shift
                PYTEST_ARGS+=("$@")
                break
                ;;
            -*)
                warn "Unknown option: $1"
                show_help
                ;;
            *)
                PYTEST_ARGS+=("$1")
                shift
                ;;
        esac
    done
    
    # Load configuration
    load_config
    
    # Initialize environment
    init_environment
    
    # Run tests and checks
    local exit_code=0
    
    # Run linters if requested
    if [[ "$RUN_LINTERS" -eq 1 ]]; then
        run_linters || exit_code=$?
    fi
    
    # Run custom test commands
    run_custom_tests || exit_code=$?
    
    # Run pytest tests
    run_pytest || exit_code=$?
    
    # Run codex-lite specific tests if requested
    if [[ "$RUN_CODEX_LITE" -eq 1 ]]; then
        run_codex_lite_tests || exit_code=$?
    fi
    
    # Print final status
    if [ $exit_code -eq 0 ]; then
        success "All tests completed successfully!"
    else
        warn "Some tests failed with exit code $exit_code"
    fi
    
    exit $exit_code
}

# Run the main function
main "$@"