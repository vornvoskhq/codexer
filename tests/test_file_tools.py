import pytest
from integration.tools import file_tools

def test_read_and_write_file(tmp_path):
    file_path = tmp_path / "foo.txt"
    file_tools.write_file(str(file_path), "bar")
    content = file_tools.read_file(str(file_path))
    assert content == "bar"

def test_list_directory(tmp_path):
    (tmp_path / "a.txt").write_text("a")
    (tmp_path / "b.txt").write_text("b")
    entries = file_tools.list_directory(str(tmp_path))
    assert "a.txt" in entries and "b.txt" in entries

def test_find_file_upwards(tmp_path):
    d1 = tmp_path / "dir1"
    d2 = d1 / "dir2"
    d2.mkdir(parents=True)
    target = d1 / "myfile.txt"
    target.write_text("target")
    result = file_tools.find_file_upwards(str(d2), "myfile.txt")
    assert result == str(target)