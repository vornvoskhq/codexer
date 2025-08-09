from integration.tools import code_analysis

def test_parse_tree_runs():
    tree = code_analysis.parse_tree("a = 1")
    assert tree

def test_semantic_analysis_runs(tmp_path):
    path = tmp_path / "foo.py"
    path.write_text("print(123)")
    result = code_analysis.semantic_analysis(str(path))
    assert isinstance(result, dict)
    assert result["file"] == str(path)