#!/usr/bin/env python3
"""
Codex CLI enhanced with agentic tools.
"""
import argparse

from integration.framework.agent import AgenticExecutor
from integration.tools import (
    file_tools,
    git_tools,
    code_analysis,
    project_tools,
    advanced_agent_tools,
    ide_tools,
    specialized_tools,
)

# mapping of tool names to callable functions
TOOLS = {
    'file_read': file_tools.read_file,
    'file_write': file_tools.write_file,
    'list_directory': file_tools.list_directory,
    'find_file_upwards': file_tools.find_file_upwards,
    'git_status': git_tools.status,
    'git_commit': git_tools.auto_commit,
    'parse_tree': code_analysis.parse_tree,
    'semantic_analysis': code_analysis.semantic_analysis,
    'detect_project_type': project_tools.detect_project_type,
    'list_dependencies': project_tools.list_dependencies,
    'plan_tasks': advanced_agent_tools.plan_tasks,
    'coordinate_agents': advanced_agent_tools.coordinate_agents,
    'lint_file': ide_tools.lint_file,
    'autocomplete_code': ide_tools.autocomplete_code,
    'refactor_extract_method': ide_tools.refactor_extract_method,
    'run_django_command': specialized_tools.run_django_command,
    'build_docker_image': specialized_tools.build_docker_image,
}

def main():
    parser = argparse.ArgumentParser(
        prog='codex-lite',
        description='Codex CLI with agentic tools'
    )
    subparsers = parser.add_subparsers(dest='command')

    # Agent command to invoke tools
    agent_parser = subparsers.add_parser(
        'agent', help='Invoke an agentic tool'
    )
    agent_parser.add_argument(
        'tool', nargs='?', choices=TOOLS.keys(),
        help='Tool to invoke (runs single command)'
    )
    agent_parser.add_argument(
        'params', nargs=argparse.REMAINDER,
        help='Positional arguments for the selected tool'
    )

    args = parser.parse_args()

    if args.command == 'agent':
        executor = AgenticExecutor()
        # Single-command mode
        if args.tool:
            tool_fn = TOOLS[args.tool]
            result = tool_fn(*args.params)
            print(result)
        else:
            # Conversational interactive mode with optional multiline input
            import os
            import readline
            from integration.framework.llm_agent import LLMAgent
            import yaml
            # Load features config
            try:
                cfg = yaml.safe_load(open('agent_config.yaml')) or {}
                multiline = cfg.get('features', {}).get('multiline_input', False)
            except Exception:
                multiline = False
            agent = LLMAgent()
            commands = list(TOOLS.keys()) + ['history', 'exit', 'quit']
            def completer(text, state):
                options = [c for c in commands if c.startswith(text)]
                return options[state] if state < len(options) else None
            readline.set_completer(completer)
            readline.parse_and_bind('tab: complete')
            history = []  # list of (user, agent) exchanges
            print("Entering chat mode (type 'history', 'exit', or 'quit')")
            while True:
                try:
                    if multiline:
                        # collect until empty line
                        lines = []
                        while True:
                            line = input()  # no prompt
                            if not line:
                                break
                            lines.append(line)
                        user_input = '\n'.join(lines).strip()
                    else:
                        user_input = input('You> ').strip()
                except (EOFError, KeyboardInterrupt):
                    print()  # newline
                    break
                cmd = user_input.lower()
                if not user_input or cmd in ('exit', 'quit'):
                    break
                if cmd == 'history':
                    for u, a in history:
                        print(f"You: {u}")
                        print(f"Agent: {a}")
                    continue
                print(f"You: {user_input}")
                try:
                    reply = agent.run(user_input)
                except Exception as e:
                    reply = f"<Error: {e}>"
                history.append((user_input, reply))
                for line in reply.splitlines():
                    print(f"Agent: {line}")
            print("Exiting chat mode.")
            tool_fn = TOOLS[args.tool]
            # All params passed positionally; convert strings if needed
            result = tool_fn(*args.params)
            print(result)
        else:
            # Interactive REPL mode
            # Map tool name to AgenticExecutor method
            AGENT_METHODS = {
                'file_read': executor.run_file_read,
                'file_write': executor.run_file_write,
                'list_directory': executor.run_directory_list,
                'find_file_upwards': executor.run_find_up,
                'git_status': executor.run_git_status,
                'git_commit': executor.run_git_commit,
                'parse_tree': code_analysis.parse_tree,
                'semantic_analysis': code_analysis.semantic_analysis,
                'detect_project_type': project_tools.detect_project_type,
                'list_dependencies': project_tools.list_dependencies,
                'plan_tasks': advanced_agent_tools.plan_tasks,
                'coordinate_agents': advanced_agent_tools.coordinate_agents,
                'lint_file': ide_tools.lint_file,
                'autocomplete_code': ide_tools.autocomplete_code,
                'refactor_extract_method': ide_tools.refactor_extract_method,
                'run_django_command': specialized_tools.run_django_command,
                'build_docker_image': specialized_tools.build_docker_image,
            }
            print("Agent interactive mode. Type 'exit' or 'quit' to leave.")
            from integration.framework.llm_agent import LLMAgent
            llm_agent = LLMAgent()
            while True:
                try:
                    inp = input("Agent> ")
                except (EOFError, KeyboardInterrupt):
                    print()
                    break
                line = inp.strip()
                if not line:
                    continue
                if line.lower() in {"exit", "quit"}:
                    break
                tokens = line.split()
                tool = tokens[0]
                params = tokens[1:]
                if tool in AGENT_METHODS:
                    method = AGENT_METHODS[tool]
                    try:
                        result = method(*params)
                    except Exception as e:
                        print(f"Error: {e}")
                else:
                    # Not a tool: run through LLM agent loop
                    try:
                        result = llm_agent.run(line)
                        print(result)
                    except Exception as e:
                        print(f"LLM Agent error: {e}")
    else:
        parser.print_help()

if __name__ == '__main__':
    main()