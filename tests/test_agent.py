import io
import os
import tempfile
import shutil
import sys
from contextlib import redirect_stdout

import pytest

from integration.framework.agent import AgenticExecutor

@pytest.fixture
def temp_file(tmp_path):
    file = tmp_path / "example.txt"
    file.write_text("initial content")
    return str(file)

@pytest.fixture
def temp_dir(tmp_path):
    d = tmp_path / "subdir"
    d.mkdir()
    (d / "foo.txt").write_text("foo")
    (d / "bar.txt").write_text("bar")
    return str(d)

def test_run_file_read(temp_file):
    agent = AgenticExecutor()
    f = io.StringIO()
    with redirect_stdout(f):
        result = agent.run_file_read(temp_file)
    output = f.getvalue()
    assert "SYSTEM: You are an agent capable of reading files." in output
    assert "TOOL_RESPONSE:" in output
    assert "initial content" in result

def test_run_file_write(tmp_path):
    agent = AgenticExecutor()
    file_path = tmp_path / "write_test.txt"
    content = "hello, agent!"
    f = io.StringIO()
    with redirect_stdout(f):
        result = agent.run_file_write(str(file_path), content)
    output = f.getvalue()
    assert os.path.exists(file_path)
    assert file_path.read_text() == content
    assert "TOOL_RESPONSE" in output

def test_run_directory_list(temp_dir):
    agent = AgenticExecutor()
    f = io.StringIO()
    with redirect_stdout(f):
        result = agent.run_directory_list(temp_dir)
    output = f.getvalue()
    assert "foo.txt" in result
    assert "bar.txt" in result
    assert "TOOL_RESPONSE" in output

def test_run_find_up(tmp_path):
    agent = AgenticExecutor()
    # create nested dirs and file
    d1 = tmp_path / "a"
    d2 = d1 / "b"
    d2.mkdir(parents=True)
    target_file = d1 / "target.txt"
    target_file.write_text("target")
    f = io.StringIO()
    with redirect_stdout(f):
        result = agent.run_find_up(str(d2), "target.txt")
    output = f.getvalue()
    assert str(target_file) == result
    assert "TOOL_RESPONSE" in output

def test_run_git_status(monkeypatch):
    agent = AgenticExecutor()
    path = "/repo"
    def fake_git_status(p):
        assert p == path
        return "On branch main\nnothing to commit"
    monkeypatch.setattr("integration.tools.git_tools.status", fake_git_status)
    f = io.StringIO()
    with redirect_stdout(f):
        result = agent.run_git_status(path)
    output = f.getvalue()
    assert "On branch main" in result
    assert "TOOL_RESPONSE" in output

def test_run_git_commit(monkeypatch):
    agent = AgenticExecutor()
    path = "/repo"
    files = ["a.txt", "b.txt"]
    def fake_git_commit(p, fs):
        assert p == path and fs == files
        return "Committed 2 files"
    monkeypatch.setattr("integration.tools.git_tools.auto_commit", fake_git_commit)
    f = io.StringIO()
    with redirect_stdout(f):
        result = agent.run_git_commit(path, files)
    output = f.getvalue()
    assert "Committed 2 files" in result
    assert "TOOL_RESPONSE" in output