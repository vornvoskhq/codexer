import os
import pytest
from integration.functions import file_system

def test_read_file_and_write_file(tmp_path):
    file_path = tmp_path / "hello.txt"
    content = "Hello, world!"
    file_system.write_file(str(file_path), content)
    read_content = file_system.read_file(str(file_path))
    assert read_content == content

def test_read_file_missing(tmp_path):
    with pytest.raises(FileNotFoundError):
        file_system.read_file(str(tmp_path / "no_such_file.txt"))

def test_write_file_overwrite(tmp_path):
    file_path = tmp_path / "file.txt"
    file_system.write_file(str(file_path), "first")
    file_system.write_file(str(file_path), "second")
    assert file_system.read_file(str(file_path)) == "second"

def test_list_directory_basic(tmp_path):
    (tmp_path / "a.txt").write_text("a")
    (tmp_path / "b.txt").write_text("b")
    entries = file_system.list_directory(str(tmp_path))
    assert set(entries) >= {"a.txt", "b.txt"}

def test_list_directory_not_a_directory(tmp_path):
    file_path = tmp_path / "file.txt"
    file_path.write_text("content")
    with pytest.raises(NotADirectoryError):
        file_system.list_directory(str(file_path))

def test_find_file_upwards_found(tmp_path):
    # Create nested structure: tmp_path/dir1/dir2/
    d1 = tmp_path / "dir1"
    d2 = d1 / "dir2"
    d2.mkdir(parents=True)
    target = d1 / "myfile.txt"
    target.write_text("target")
    result = file_system.find_file_upwards(str(d2), "myfile.txt")
    assert result == str(target)

def test_find_file_upwards_not_found(tmp_path):
    d1 = tmp_path / "dir1"
    d2 = d1 / "dir2"
    d2.mkdir(parents=True)
    result = file_system.find_file_upwards(str(d2), "nope.txt")
    assert result is None

def test_find_file_upwards_at_root(monkeypatch, tmp_path):
    # Simulate being at the filesystem root
    root = tmp_path
    (root / "target.txt").write_text("x")
    monkeypatch.chdir(str(root))
    result = file_system.find_file_upwards(str(root), "target.txt")
    assert result == str(root / "target.txt")