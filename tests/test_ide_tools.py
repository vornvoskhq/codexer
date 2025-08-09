from integration.tools import ide_tools

def test_lint_file(tmp_path):
    file = tmp_path / "test.py"
    file.write_text("x=1")
    issues = ide_tools.lint_file(str(file))
    assert isinstance(issues, list)

def test_autocomplete_code():
    completions = ide_tools.autocomplete_code("pri")
    assert any("pri" in c for c in completions)

def test_refactor_extract_method():
    code = "def foo():\n    pass"
    out = ide_tools.refactor_extract_method(code, 1, 2)
    assert "Extracted method" in out