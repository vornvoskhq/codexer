from codex.llm.openrouter import send_initial_prompt, query
from codex.cli.tools import patcher, shell_suggester
from rich.console import Console
from codex.cli.config import load_config

console = Console()
#cfg = load_config()
#console.print(send_initial_prompt())  # primes the session

# Display system prompt response at session start
system_intro = query("system: begin session")
console.print("[bold cyan]Initial System (main.py):[/bold cyan] " + system_intro)

console.print("[bold green]Codex Lite CLI[/bold green] - type 'exit' to quit.")

while True:
    user_input = input(">>> ").strip()
    if user_input == "exit": break
    if user_input.startswith("patch "):
        patcher.patch_file("target.py", user_input[6:])
    elif user_input.startswith("shell "):
        shell_suggester.suggest_shell_command(user_input[6:])
    else:
        response = query(user_input)
        console.print(response)
