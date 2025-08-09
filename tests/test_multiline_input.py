import builtins
import os
import yaml
import pytest

from codex_lite.main import main, TOOLS

class DummyAgent:
    def __init__(self):
        self.last_prompt = None
    def run(self, prompt):
        self.last_prompt = prompt
        return f"Got:{prompt}"

@pytest.fixture(autouse=True)
def setup_agent(monkeypatch):
    # Replace LLMAgent with DummyAgent
    monkeypatch.setattr('integration.framework.llm_agent.LLMAgent', lambda: DummyAgent())
    yield

def write_config(tmp_path, multiline):
    cfg = {'features': {'multiline_input': multiline}}
    conf = tmp_path / 'agent_config.yaml'
    conf.write_text(yaml.dump(cfg))
    return conf

@pytest.mark.parametrize('multiline,inputs,expected', [
    (True, ['line1', 'line2', '', 'exit'], 'line1\nline2'),
    (False, ['one-liner', 'exit'], 'one-liner'),
])
def test_multiline_flag(monkeypatch, tmp_path, capsys, multiline, inputs, expected):
    # Setup config file in cwd
    conf = write_config(tmp_path, multiline)
    monkeypatch.chdir(tmp_path)
    # Simulate CLI call
    monkeypatch.setenv('HOME', str(tmp_path))
    monkeypatch.setattr('sys.argv', ['codex-lite', 'agent'])
    # Prepare input sequence
    inp_iter = iter(inputs)
    def fake_input(prompt=''):
        return next(inp_iter)
    monkeypatch.setattr(builtins, 'input', fake_input)
    # Run main
    main_module.main()
    out = capsys.readouterr().out
    # Check agent.run received expected combined prompt
    # DummyAgent stores last_prompt
    # Ensure output shows Got:<expected>
    assert f"Agent: Got:{expected}" in out