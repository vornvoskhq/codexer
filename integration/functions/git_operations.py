"""
Legacy module for git operations. This is now a wrapper around integration.tools.git_tools.
New code should import directly from integration.tools.git_tools instead.
"""
import warnings
from typing import Dict, Any, List

# Import from git_tools
from integration.tools.git_tools import status as git_status, auto_commit as git_auto_commit

# Issue deprecation warning
warnings.warn(
    "The 'integration.functions.git_operations' module is deprecated. "
    "Please use 'integration.tools.git_tools' instead.",
    DeprecationWarning,
    stacklevel=2
)

# Re-export functions for backward compatibility
status = git_status
auto_commit = git_auto_commit

# This file is maintained for backward compatibility.
# New code should import directly from integration.tools.git_tools instead.