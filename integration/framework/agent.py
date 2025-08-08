"""
AgenticExecutor: An integration framework simulating agentic orchestration of file and git operations.

This framework demonstrates how an LLM agent (e.g., glm-4.5-air) can orchestrate classic developer tasks
by invoking modular functions for file and git operations. Each method simulates an agent loop, printing
a SYSTEM prompt, USER request, TOOL invocation, and TOOL_RESPONSE to mimic a conversational workflow.

Each method calls directly into the functions modules (assumed to be importable), 
prints the agentic message sequence, and returns the underlying result for testability.

Available methods:
    - run_file_read(path)
    - run_file_write(path, content)
    - run_directory_list(path)
    - run_find_up(path, filename)
    - run_git_status(path)
    - run_git_commit(path, files)
"""

from integration.framework import functions_fileops
from integration.framework import functions_git

class AgenticExecutor:
    """
    AgenticExecutor simulates an LLM agent orchestrating file and git operations.

    Each method triggers a message sequence:
    - SYSTEM: Sets context for the LLM agent.
    - USER: States the user's request.
    - TOOL: Shows the tool/function call with parameters.
    - TOOL_RESPONSE: Shows the tool's output/result.

    Methods return the TOOL_RESPONSE, allowing verification in tests.
    """

    def run_file_read(self, path):
        print("SYSTEM: You are an agent capable of reading files.")
        print(f"USER: Please read the contents of the file at '{path}'.")
        print(f"TOOL: file_read(path='{path}')")
        result = functions_fileops.file_read(path)
        print(f"TOOL_RESPONSE: {repr(result)}")
        return result

    def run_file_write(self, path, content):
        print("SYSTEM: You are an agent capable of writing to files.")
        print(f"USER: Please write the following content to '{path}': {repr(content)}")
        print(f"TOOL: file_write(path='{path}', content={repr(content)})")
        result = functions_fileops.file_write(path, content)
        print(f"TOOL_RESPONSE: {repr(result)}")
        return result

    def run_directory_list(self, path):
        print("SYSTEM: You are an agent capable of listing directory contents.")
        print(f"USER: Please list the contents of the directory '{path}'.")
        print(f"TOOL: directory_list(path='{path}')")
        result = functions_fileops.directory_list(path)
        print(f"TOOL_RESPONSE: {repr(result)}")
        return result

    def run_find_up(self, path, filename):
        print("SYSTEM: You are an agent capable of finding files up the directory tree.")
        print(f"USER: Please find the file named '{filename}' starting from '{path}' upwards.")
        print(f"TOOL: find_up(path='{path}', filename='{filename}')")
        result = functions_fileops.find_up(path, filename)
        print(f"TOOL_RESPONSE: {repr(result)}")
        return result

    def run_git_status(self, path):
        print("SYSTEM: You are an agent capable of checking git status.")
        print(f"USER: Please show the git status for the repository at '{path}'.")
        print(f"TOOL: git_status(path='{path}')")
        result = functions_git.git_status(path)
        print(f"TOOL_RESPONSE: {repr(result)}")
        return result

    def run_git_commit(self, path, files):
        print("SYSTEM: You are an agent capable of committing files to git.")
        print(f"USER: Please commit the following files in '{path}': {files}")
        print(f"TOOL: git_commit(path='{path}', files={files})")
        result = functions_git.git_commit(path, files)
        print(f"TOOL_RESPONSE: {repr(result)}")
        return result