import os
import pytest
import types
from integration.framework.llm_agent import LLMAgent

class DummyExecutor:
    def run_file_read(self, path):
        return f"read:{path}"

def test_llm_agent_run_tool(monkeypatch):
    # Patch OpenRouter call and AgenticExecutor
    os.environ["OPENROUTER_API_KEY"] = "test-key"
    agent = LLMAgent(api_key="test-key")
    agent._send_to_llm = lambda messages: "TOOL: file_read(test.txt)"
    agent._execute_tool = lambda name, args: "file content"
    # Next call returns final answer
    responses = iter([
        "TOOL: file_read(test.txt)",
        "FINAL ANSWER: done"
    ])
    def fake_send(messages):
        return next(responses)
    agent._send_to_llm = fake_send
    agent._execute_tool = lambda name, args: "mocked output"
    result = agent.run("Read file please")
    assert result == "done"

def test_llm_agent_final_only(monkeypatch):
    os.environ["OPENROUTER_API_KEY"] = "test-key"
    agent = LLMAgent(api_key="test-key")
    agent._send_to_llm = lambda messages: "FINAL ANSWER: just done"
    result = agent.run("Say hi")
    assert result == "just done"

def test_llm_agent_tool_parse():
    agent = LLMAgent(api_key="test-key")
    text = "TOOL: file_read(foo.txt)"
    assert agent._parse_tool_call(text) == ('file_read', 'foo.txt')
    final = "FINAL ANSWER: hello"
    assert agent._parse_final_answer(final) == 'hello'