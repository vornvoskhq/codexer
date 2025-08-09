import os
import requests
import re
from typing import Optional

from integration.framework.agent import AgenticExecutor

# Import TOOLS mapping from codex-lite
import sys
import importlib

codex_main = importlib.import_module("codex-lite.main".replace("-", "_"))
TOOLS = codex_main.TOOLS

class LLMAgent:
    """
    An LLM-driven agent that interacts with glm-4.5-air via OpenRouter, orchestrating tool calls
    using the AgenticExecutor. The agent loop continues until the LLM provides a final answer.

    Methods:
        - __init__(self, api_key: Optional[str] = None)
        - run(self, prompt: str) -> str
    """

    def __init__(self, api_key: Optional[str] = None):
        """
        Initialize the LLMAgent.

        Args:
            api_key (Optional[str]): The OpenRouter API key. If not provided, read from
                                     environment variable 'OPENROUTER_API_KEY'.
        """
        self.api_key = api_key or os.getenv("OPENROUTER_API_KEY")
        if not self.api_key:
            raise ValueError("OpenRouter API key must be set in OPENROUTER_API_KEY env variable.")
        self.executor = AgenticExecutor()
        self.tools_doc = self._build_tools_doc()

    def _build_tools_doc(self) -> str:
        """
        Build a documentation string for available tools and their signatures.

        Returns:
            str: Multiline string listing all available tools.
        """
        import inspect
        lines = []
        for tool_name, fn in TOOLS.items():
            try:
                sig = str(inspect.signature(fn))
            except (ValueError, TypeError):
                continue
            lines.append(f"{tool_name}{sig}")
        return "\n".join(lines)

    def _build_system_msg(self) -> str:
        """
        Build a SYSTEM message describing the agent's capabilities and tools.

        Returns:
            str: The system message for the LLM.
        """
        return (
            "You are a fully autonomous coding agent. "
            "You can read, write, list, find, and modify files in the current directory. "
            "Use these tools to edit the codebase and persist changes.\n"
            "You have access to these tools:\n"
            f"{self.tools_doc}\n"
            "To use a tool, respond with TOOL: <tool_name>(<args>) on a line by itself.\n"
            "When you are done, reply with FINAL ANSWER: <your answer>."
        )

    def _send_to_llm(self, messages):
        """
        Send the conversation to OpenRouter LLM and return the response text.

        Args:
            messages (list): List of dicts with 'role' and 'content'.

        Returns:
            str: The LLM's response.
        """
        url = "https://openrouter.ai/api/v1/chat/completions"
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "HTTP-Referer": "http://localhost",
            "X-Title": "Codex Agent CLI",
            "Content-Type": "application/json",
        }
        payload = {
            "model": "glm-4.5-air",
            "messages": messages,
            "stream": False,
        }
        resp = requests.post(url, headers=headers, json=payload, timeout=60)
        resp.raise_for_status()
        data = resp.json()
        return data["choices"][0]["message"]["content"]

    def _parse_tool_call(self, text: str):
        """
        Parse the LLM response for a tool call in the format TOOL: <tool_name>(<args>).

        Returns:
            tuple (tool_name, args_str) or None.
        """
        m = re.search(r"TOOL:\s*(\w+)\((.*?)\)", text, re.DOTALL)
        if m:
            return m.group(1), m.group(2)
        return None

    def _parse_final_answer(self, text: str) -> Optional[str]:
        """
        Parse for a final answer marker 'FINAL ANSWER:'.

        Returns:
            str or None
        """
        m = re.search(r"FINAL ANSWER:\s*(.+)", text, re.DOTALL)
        if m:
            return m.group(1).strip()
        return None

    def _execute_tool(self, tool_name: str, args_str: str):
        """
        Execute the tool using AgenticExecutor if available, else raw TOOLS mapping.

        Args:
            tool_name (str): Name of the tool (e.g. 'file_read')
            args_str (str): Argument string from LLM (comma or space separated)

        Returns:
            str: Output from the tool
        """
        # Dynamically construct method_map
        method_map = {}
        for name in TOOLS:
            exec_method = getattr(self.executor, f"run_{name}", None)
            if callable(exec_method):
                method_map[name] = exec_method
            else:
                method_map[name] = TOOLS[name]

        # parse args (naively split by comma if present, else space)
        arglist = []
        if args_str:
            if ',' in args_str:
                arglist = [a.strip() for a in args_str.split(',') if a.strip()]
            else:
                arglist = [a.strip() for a in args_str.split() if a.strip()]

        if tool_name in method_map:
            return method_map[tool_name](*arglist)
        else:
            raise ValueError(f"Unknown tool: {tool_name}")

    def run(self, prompt: str) -> str:
        """
        Run the agentic LLM loop, sending the prompt to the LLM, handling tool calls,
        executing tools, and looping until a final answer is produced.

        Args:
            prompt (str): The user's problem or request.

        Returns:
            str: The final answer from the LLM.
        """
        messages = []
        sys_msg = self._build_system_msg()
        messages.append({"role": "system", "content": sys_msg})
        messages.append({"role": "user", "content": prompt})

        while True:
            llm_response = self._send_to_llm(messages)
            tool_call = self._parse_tool_call(llm_response)
            final = self._parse_final_answer(llm_response)
            if tool_call:
                tool_name, args_str = tool_call
                try:
                    tool_output = self._execute_tool(tool_name, args_str)
                except Exception as e:
                    tool_output = f"Tool error: {e}"
                messages.append({"role": "assistant", "content": llm_response})
                messages.append({"role": "user", "content": f"TOOL_RESPONSE: {tool_output}"})
            elif final:
                return final
            else:
                messages.append({"role": "assistant", "content": llm_response})