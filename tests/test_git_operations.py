import os
import shutil
import tempfile

import pytest
from integration.tools.git_tools import status, auto_commit

def write_file(path, text):
    with open(path, "w") as f:
        f.write(text)

def test_git_status_and_commit(tmp_path):
    # Setup: create a new git repo
    repo_dir = tmp_path / "repo"
    repo_dir.mkdir()
    os.chdir(repo_dir)
    os.system("git init")

    # Create a file and check status
    file1 = repo_dir / "file1.txt"
    write_file(file1, "hello world")
    s1 = status(str(repo_dir))
    assert "file1.txt" in s1["untracked"]
    assert not s1["modified"]
    assert not s1["staged"]

    # Stage and commit using auto_commit
    sha = auto_commit(str(repo_dir), ["file1.txt"], ai_message=False)
    assert isinstance(sha, str) and len(sha) in (40, 7)

    # No changes now
    s2 = status(str(repo_dir))
    assert not s2["untracked"]
    assert not s2["modified"]
    assert not s2["staged"]

    # Modify file1.txt (unstaged)
    write_file(file1, "new contents")
    s3 = status(str(repo_dir))
    assert "file1.txt" in s3["modified"]
    assert not s3["staged"]
    assert not s3["untracked"]

    # Stage but don't commit
    os.system(f"git add {file1}")
    s4 = status(str(repo_dir))
    assert "file1.txt" in s4["staged"]
    assert "file1.txt" not in s4["modified"]

    # Add another file (untracked)
    file2 = repo_dir / "file2.txt"
    write_file(file2, "second file")
    s5 = status(str(repo_dir))
    assert "file2.txt" in s5["untracked"]

    # Commit both files using auto_commit with ai_message=True
    sha2 = auto_commit(str(repo_dir), ["file1.txt", "file2.txt"], ai_message=True)
    assert isinstance(sha2, str)

    # After commit, both files should be clean
    s6 = status(str(repo_dir))
    assert not s6["untracked"]
    assert not s6["modified"]
    assert not s6["staged"]