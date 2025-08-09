from integration.tools import project_tools

def test_detect_project_type():
    assert project_tools.detect_project_type("setup.py") == "python"

def test_list_dependencies():
    result = project_tools.list_dependencies("some/path")
    assert isinstance(result, list)
    assert "pytest" in result