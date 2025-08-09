import os
import re
from typing import Optional
from codex.llm.openrouter import query as openrouter_query

from integration.framework.agent import AgenticExecutor

# Import TOOLS mapping from codex-lite
import sys
import importlib

codex_main = importlib.import_module("codex_lite.main")
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
        # Load tool config
        import yaml
        try:
            cfg = yaml.safe_load(open('agent_tools_config.yaml')) or {}
            self.tool_enabled = {k: v for k, v in cfg.get('tools', {}).items()}
        except Exception:
            # default all enabled
            self.tool_enabled = {name: True for name in TOOLS}
        self.tools_doc = self._build_tools_doc()
        # Load system prompts from config
        import toml
        cfg_path = os.path.expanduser('~/.codex/config.toml')
        try:
            cfg = toml.loads(open(cfg_path).read())
            sp = cfg.get('system_prompt')
            spf = cfg.get('system_prompt_file')
            # read long prompt
            if spf:
                self.long_system_prompt = open(os.path.expanduser(spf)).read()
            else:
                self.long_system_prompt = sp or ''
            # short fallback
            self.short_system_prompt = sp or ''
        except Exception:
            self.long_system_prompt = ''
            self.short_system_prompt = ''
        self._first_run = True
        # Load system prompts from config
        import toml
        cfg_path = os.path.expanduser('~/.codex/config.toml')
        try:
            cfg = toml.loads(open(cfg_path).read())
            sp = cfg.get('system_prompt')
            spf = cfg.get('system_prompt_file')
            # read long prompt
            if spf:
                self.long_system_prompt = open(os.path.expanduser(spf)).read()
            else:
                self.long_system_prompt = sp or ''
            # short fallback
            self.short_system_prompt = sp or ''
        except Exception:
            self.long_system_prompt = ''
            self.short_system_prompt = ''
        self._first_run = True

    def _build_tools_doc(self) -> str:
        """
        Build a documentation string for available tools and their signatures.

        Returns:
            str: Multiline string listing all available tools.
        """
        import inspect
        lines = []
        for tool_name, fn in TOOLS.items():
            if not self.tool_enabled.get(tool_name, True):
            raise ValueError(f"Tool '{tool_name}' is disabled by configuration.")
        if tool_name in method_map:
            return method_map[tool_name](*arglist)
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
        # Use long system prompt on first run, else default
        sys_msg = self.long_system_prompt if self._first_run and self.long_system_prompt else self._build_system_msg()
        self._first_run = False
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

    def _send_to_llm(self, messages):
        """
        Send the conversation to OpenRouter LLM and return the response text.

        Args:
            messages (list): List of dicts with 'role' and 'content'.

        Returns:
            str: The LLM's response.
        """
        return openrouter_query(messages, api_key=self.api_key, model='glm-4.5-air', stream=False)