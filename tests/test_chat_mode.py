import builtins
import io
import pytest

from codex_lite.main import main, TOOLS

class DummyAgent:
    def __init__(self): pass
    def run(self, prompt):
        return f"Echo: {prompt}"

@pytest.fixture(autouse=True)
def dummy_agent(monkeypatch):
    monkeypatch.setattr('integration.framework.llm_agent.LLMAgent', lambda: DummyAgent())
    yield

def test_chat_basic_exit(monkeypatch, capsys):
    inputs = iter(['hello world', 'exit'])
    monkeypatch.setattr(builtins, 'input', lambda prompt='': next(inputs))
    # capture print
    main_module.TOOLS = {}  # no tools for completion
    main_module.main_args = ['agent']
    # simulate args
    monkeypatch.setattr('sys.argv', ['codex-lite', 'agent'])
    main_module.main()
    out = capsys.readouterr().out
    assert 'Entering chat mode' in out
    assert 'You: hello world' in out
    assert 'Agent: Echo: hello world' in out
    assert 'Exiting chat mode.' in out

def test_chat_history(monkeypatch, capsys):
    inputs = iter(['first', 'second', 'history', 'quit'])
    monkeypatch.setattr(builtins, 'input', lambda prompt='': next(inputs))
    main_module.TOOLS = {'dummy': None}
    monkeypatch.setattr('sys.argv', ['codex-lite', 'agent'])
    main_module.main()
    out = capsys.readouterr().out
    # history should show first two exchanges
    assert 'You: first' in out
    assert 'Agent: Echo: first' in out
    assert 'You: second' in out
    assert 'Agent: Echo: second' in out
    # history block
    # ensure that after typing history, we see repeats
    history_section = out.split('history')[-1]
    assert 'You: first' in history_section
    assert 'Agent: Echo: first' in history_section
    assert 'You: second' in history_section
    assert 'Agent: Echo: second' in history_section