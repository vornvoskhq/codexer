import os
from integration.tools.git_tools import status, auto_commit

def write_file(path, text):
    with open(path, "w") as f:
        f.write(text)

def test_git_status_and_commit(tmp_path):
    repo_dir = tmp_path / "repo"
    repo_dir.mkdir()
    os.chdir(repo_dir)
    os.system("git init")

    file1 = repo_dir / "file1.txt"
    write_file(file1, "hello world")
    s1 = status(str(repo_dir))
    assert "file1.txt" in s1["untracked"]

    sha = auto_commit(str(repo_dir), ["file1.txt"], ai_message=False)
    assert isinstance(sha, str)