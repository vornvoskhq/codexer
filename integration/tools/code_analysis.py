"""
Phase 3: Code analysis tools.
"""

from typing import Any, Dict

import ast

def parse_tree(source_code: str) -> Any:
    """
    Parse the given source code into an AST.

    Args:
        source_code (str): Source code as string.

    Returns:
        Any: Abstract Syntax Tree (AST) object.
    """
    return ast.parse(source_code)

def semantic_analysis(file_path: str) -> Dict[str, Any]:
    """
    Perform a stub semantic analysis for the specified file.

    Args:
        file_path (str): Path to a source code file.

    Returns:
        Dict[str, Any]: Dictionary with semantic information (stub).
    """
    # Stub: in a real implementation, analyze symbols, types, etc.
    return {"file": file_path, "analysis": "success", "symbols": [], "types": []}