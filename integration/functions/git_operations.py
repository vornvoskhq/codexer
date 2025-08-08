from typing import Dict, Any, List
import os

from git import Repo, InvalidGitRepositoryError, GitCommandError, NoSuchPathError


def status(repo_path: str) -> Dict[str, Any]:
    """
    Get the status of a Git repository, returning lists of modified, untracked, and staged files.

    Args:
        repo_path (str): Path to the root of the git repository.

    Returns:
        Dict[str, Any]: Dictionary with keys:
            - 'modified': list of modified (unstaged) file paths.
            - 'untracked': list of untracked file paths.
            - 'staged': list of staged file paths.

    Raises:
        ValueError: If the provided path is not a valid git repository.
    """
    try:
        repo = Repo(repo_path)
    except (InvalidGitRepositoryError, NoSuchPathError) as e:
        raise ValueError(f"Invalid git repository at {repo_path}: {e}")

    # Untracked files
    untracked = list(repo.untracked_files)

    # Staged files (diff between index and HEAD)
    staged = [item.a_path for item in repo.index.diff("HEAD") if item.change_type != "D"]

    # Modified files (unstaged)
    modified = [item.a_path for item in repo.index.diff(None) if item.change_type != "D"]

    return {"modified": modified, "untracked": untracked, "staged": staged}


def auto_commit(repo_path: str, files: List[str], ai_message: bool = True) -> str:
    """
    Stage the given files and commit them with an AI-generated message placeholder or a default message.

    Args:
        repo_path (str): Path to the root of the git repository.
        files (List[str]): List of file paths (relative to repo root) to stage and commit.
        ai_message (bool): If True, use an AI message placeholder. If False, use a default message.

    Returns:
        str: The commit SHA of the new commit.

    Raises:
        ValueError: If commit fails or repo is invalid.
    """
    try:
        repo = Repo(repo_path)
    except (InvalidGitRepositoryError, NoSuchPathError) as e:
        raise ValueError(f"Invalid git repository at {repo_path}: {e}")

    # Stage files
    abs_files = [os.path.join(repo.working_tree_dir, f) for f in files]
    try:
        repo.index.add(abs_files)
    except GitCommandError as e:
        raise ValueError(f"Failed to add files to git index: {e}")

    # Create commit message
    if ai_message:
        commit_msg = "[AI] Commit message would go here"
    else:
        commit_msg = "Auto-commit changes"

    # Commit
    try:
        commit_obj = repo.index.commit(commit_msg)
    except Exception as e:
        raise ValueError(f"Git commit failed: {e}")

    return commit_obj.hexsha