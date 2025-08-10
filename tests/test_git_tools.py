import os
from integration.tools import git_tools

def write_file(path, text):
    with open(path, "w") as f:
        f.write(text)

def test_git_status_and_commit(tmp_path):
    # Setup: create a new git repo
    repo_dir = tmp_path / "repo"
    repo_dir.mkdir()
    os.chdir(repo_dir)
    
    # Initialize git repo with an initial commit
    os.system("git init")
    os.system("git config user.email 'test@example.com'")
    os.system("git config user.name 'Test User'")
    os.system("git commit --allow-empty -m 'Initial commit'")

    # Create a file and check status
    file1 = repo_dir / "file1.txt"
    write_file(file1, "hello world")
    
    # Check status after file creation
    s1 = git_tools.status(str(repo_dir))
    assert "file1.txt" in s1["untracked"]
    assert not s1["modified"]
    assert not s1["staged"]

    # Stage and commit using auto_commit
    sha = git_tools.auto_commit(str(repo_dir), ["file1.txt"], ai_message=False)
    assert isinstance(sha, str) and len(sha) in (40, 7)