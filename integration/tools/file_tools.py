import os
from typing import List, Optional

def read_file(path: str) -> str:
    """
    Read the contents of a file.

    Args:
        path (str): The path to the file.

    Returns:
        str: The content of the file as a string.

    Raises:
        FileNotFoundError: If the file does not exist.
        IOError: If the file cannot be read.
    """
    with open(path, "r", encoding="utf-8") as f:
        return f.read()

def write_file(path: str, content: str) -> None:
    """
    Write content to a file, overwriting if it exists.

    Args:
        path (str): The path to the file.
        content (str): The content to write.

    Returns:
        None

    Raises:
        IOError: If the file cannot be written.
    """
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)

def list_directory(path: str) -> List[str]:
    """
    List all entries in a directory.

    Args:
        path (str): The path to the directory.

    Returns:
        List[str]: List of entries (files and directories) in the directory.

    Raises:
        NotADirectoryError: If the path is not a directory.
        FileNotFoundError: If the directory does not exist.
    """
    return os.listdir(path)

def find_file_upwards(start_path: str, filename: str) -> Optional[str]:
    """
    Search for a file named `filename`, starting from `start_path` and moving upwards
    through parent directories until found or root is reached.

    Args:
        start_path (str): Directory to start the search from.
        filename (str): Name of the file to find.

    Returns:
        Optional[str]: The absolute path to the found file, or None if not found.
    """
    current = os.path.abspath(start_path)
    while True:
        candidate = os.path.join(current, filename)
        if os.path.isfile(candidate):
            return candidate
        parent = os.path.dirname(current)
        if parent == current:
            break
        current = parent
    return None