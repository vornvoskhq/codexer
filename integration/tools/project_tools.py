"""
Phase 4: Project tools.
"""

from typing import List

def detect_project_type(path: str) -> str:
    """
    Detect the type of project in the given path.

    Args:
        path (str): Path to inspect.

    Returns:
        str: Project type detected (stub).
    """
    # Stub: in a real implementation, look for pyproject.toml, package.json, etc.
    return "python" if "py" in path else "unknown"

def list_dependencies(path: str) -> List[str]:
    """
    List the dependencies for the project at the given path.

    Args:
        path (str): Path to the project.

    Returns:
        List[str]: List of dependency names (stub).
    """
    # Stub: real implementation would parse requirements.txt, pyproject.toml, etc.
    return ["pytest", "numpy"]