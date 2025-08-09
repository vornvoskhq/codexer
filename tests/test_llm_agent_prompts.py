import os
import tempfile
import toml
import pytest

from integration.framework.llm_agent import LLMAgent

class DummyAgenticExecutor:
    def __init__(self):
        self.calls = []
    def run_file_read(self, path):
        return f"read:{path}"

@pytest.fixture(autouse=True)
def dummy_executor(monkeypatch):
    # Patch AgenticExecutor to avoid real HTTP or file ops
    from integration.framework.llm_agent import AgenticExecutor
    monkeypatch.setattr('integration.framework.llm_agent.AgenticExecutor', lambda: DummyAgenticExecutor())
    yield

@pytest.fixture
def config_file(monkeypatch, tmp_path):
    # Create fake config file
    cfg = {
        'system_prompt': 'Short prompt',
        'system_prompt_file': str(tmp_path / 'long.txt')
    }
    long_txt = tmp_path / 'long.txt'
    long_txt.write_text('Long prompt content')
    config_dir = tmp_path / '.codex'
    config_dir.mkdir()
    cfg_file = config_dir / 'config.toml'
    cfg_file.write_text(toml.dumps(cfg))
    # Monkeypatch home
    monkeypatch.setenv('HOME', str(tmp_path))
    return cfg

def test_long_then_short_prompt(monkeypatch, config_file):
    # Ensure llm returns immediate final answer regardless
    called = {'count': 0}
    def fake_send(messages):
        # Inspect first message for long prompt
        content = messages[0]['content']
        if called['count'] == 0:
            assert 'Long prompt content' in content
        else:
            assert 'Short prompt' in content or 'you can read, write' in content
        called['count'] += 1
        # Always return final answer
        return 'FINAL ANSWER: done'
    monkeypatch.setattr('integration.framework.llm_agent.LLMAgent._send_to_llm', fake_send)
    agent = LLMAgent(api_key='x')
    res1 = agent.run('test')
    assert res1 == 'done'
    res2 = agent.run('test2')
    assert res2 == 'done'

def test_no_config_defaults(monkeypatch, tmp_path):
    # No config file, fall back to built-in prompt
    monkeypatch.setenv('HOME', str(tmp_path))
    called = {'count': 0}
    def fake_send(messages):
        content = messages[0]['content']
        # always uses built-in doc
        assert 'You are a fully autonomous coding agent' in content
        called['count'] += 1
        return 'FINAL ANSWER: ok'
    monkeypatch.setattr('integration.framework.llm_agent.LLMAgent._send_to_llm', fake_send)
    agent = LLMAgent(api_key='x')
    assert agent.run('abc') == 'ok'
    assert called['count'] == 1
    # second run uses short fallback built-in
    assert agent.run('def') == 'ok'
    assert called['count'] == 2