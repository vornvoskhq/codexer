#!/bin/bash

set -e

echo "[*] Checking Python environment..."

PYTHON_BIN=$(command -v python3 || command -v python || pyenv which python || true)
if [[ -z "$PYTHON_BIN" ]]; then
    echo "[!] No Python interpreter found. Please install Python 3."
    exit 1
fi
echo "[+] Using Python: $PYTHON_BIN"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(realpath "$SCRIPT_DIR")"
INSTALL_DIR="/usr/local/bin"
CODENAME="codex"
GLOBAL_BIN="$INSTALL_DIR/$CODENAME"

if [[ "$1" == "uninstall" ]]; then
    echo "[*] Uninstalling Codex Lite..."
    if [[ -L "$GLOBAL_BIN" || -f "$GLOBAL_BIN" ]]; then
        echo "[*] Removing existing global symlink..."
        sudo rm -f "$GLOBAL_BIN"
    fi
    echo "[+] Uninstall complete."
    exit 0
fi

echo "[*] Cloning Codex CLI..."
if [[ -d codex-lite ]]; then
    if [[ -n "$(ls -A codex-lite)" ]]; then
        if [[ "$1" == "--force" ]]; then
            echo "[*] --force specified. Removing existing codex-lite..."
            rm -rf codex-lite
            git clone https://github.com/openai/codex.git codex-lite
        else
            echo "[!] codex-lite already exists and is not empty. Use './install_codex.sh --force' to reinstall."
            exit 1
        fi
    else
        echo "[*] codex-lite exists but is empty. Removing and recloning..."
        rm -rf codex-lite
        git clone https://github.com/openai/codex.git codex-lite
    fi
else
    git clone https://github.com/openai/codex.git codex-lite
fi

cd codex-lite || { echo "[!] Failed to enter codex-lite directory"; exit 1; }

echo "[*] Creating virtual environment..."
$PYTHON_BIN -m venv .venv
source .venv/bin/activate

echo "[*] Installing minimal dependencies..."
pip install --upgrade pip
pip install requests toml rich

echo "[*] Creating source directories..."
mkdir -p codex/llm codex/cli/tools codex/cli/config codex/cli/logger logs

# Create __init__.py files to make directories proper Python packages
touch codex/__init__.py
touch codex/llm/__init__.py
touch codex/cli/__init__.py
touch codex/cli/tools/__init__.py
touch codex/cli/config/__init__.py
touch codex/cli/logger/__init__.py

# Store the absolute path to the project
CODEX_LITE_ROOT="$PROJECT_ROOT/codex-lite"

echo "[*] Creating CLI wrapper..."
cat > codex/cli/codex <<EOF
#!/bin/bash
# Resolve the absolute path to this script
SCRIPT_PATH="\$(readlink -f "\${BASH_SOURCE[0]}")"
SCRIPT_DIR="\$(dirname "\$SCRIPT_PATH")"

# Navigate to the codex-lite root directory
CODEX_ROOT="\$(dirname "\$(dirname "\$SCRIPT_DIR")")"
VENV_PYTHON="\$CODEX_ROOT/.venv/bin/python3"

if [ ! -x "\$VENV_PYTHON" ]; then
  echo "Error: Python executable not found at \$VENV_PYTHON"
  echo "Debug: SCRIPT_PATH=\$SCRIPT_PATH"
  echo "Debug: SCRIPT_DIR=\$SCRIPT_DIR"
  echo "Debug: CODEX_ROOT=\$CODEX_ROOT"
  exit 1
fi

# Add the codex-lite root to Python path so imports work
export PYTHONPATH="\$CODEX_ROOT:\$PYTHONPATH"
exec "\$VENV_PYTHON" "\$SCRIPT_DIR/main.py" "\$@"
EOF
chmod +x codex/cli/codex

echo "[*] Linking CLI globally..."
if [[ -e "$GLOBAL_BIN" || -L "$GLOBAL_BIN" ]]; then
    sudo rm -f "$GLOBAL_BIN"
fi

LAUNCHER="$PROJECT_ROOT/codex-lite/codex/cli/codex"
if [ -f "$LAUNCHER" ]; then
  sudo ln -sf "$LAUNCHER" "$GLOBAL_BIN"
else
  echo "Error: Launcher script not found at $LAUNCHER"
  exit 1
fi


echo "[*] Ensuring ~/.codex config directory exists..."
CONFIG_DIR="$HOME/.codex"
CONFIG_PATH="$CONFIG_DIR/config.toml"

mkdir -p "$CONFIG_DIR"

# Always overwrite config to ensure it's properly formatted
echo "[*] Writing config to $CONFIG_PATH..."
cat > "$CONFIG_PATH" <<'EOF'
api_key = "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"
model = "glm-4.5-air"
base_url = "https://cc.yovy.app"
system_prompt = """You are Codex, a terse CLI-native coding assistant.

Your role is to:
- Interpret user intent from short CLI inputs
- Suggest shell commands when appropriate
- Propose file patches using Python syntax
- Avoid verbose explanations unless asked
- Respond in plain text or code blocks only

If the input starts with 'patch', generate a Python code patch.
If the input starts with 'shell', suggest a shell command.
Otherwise, answer concisely with actionable code or insight.

When confused or have suggestions, you may ask questions. Never explain unless explicitly prompted."""
EOF
echo "[+] Config written."

echo "[*] Writing OpenRouter backend..."
cat > codex/llm/openrouter.py <<EOF
import requests, json
from codex.cli.config import load_config
from codex.cli.logger import log_usage

def query(prompt):
    cfg = load_config()
    headers = {
        "Authorization": f"Bearer {cfg['api_key']}",
        "Content-Type": "application/json"
    }
    payload = {
        "model": cfg['model'],
        "messages": [
            {"role": "system", "content": cfg['system_prompt']},
            {"role": "user", "content": prompt}
        ]
    }
    url = f"{cfg.get('base_url', 'https://openrouter.ai')}/api/v1/chat/completions"
    res = requests.post(url, headers=headers, data=json.dumps(payload))
    data = res.json()
    usage = data.get("usage", {})
    log_usage(cfg['model'], usage.get("total_tokens", 0), usage.get("prompt_tokens", 0), usage.get("completion_tokens", 0))
    return data["choices"][0]["message"]["content"]
EOF

echo "[*] Writing config loader..."
cat > codex/cli/config/__init__.py <<EOF
import os, toml

def load_config():
    path = os.path.expanduser("~/.codex/config.toml")
    if not os.path.exists(path):
        return {
            "model": "glm-4.5-air",
            "system_prompt": "You are a terse CLI assistant.",
            "api_key": "YOUR_OPENROUTER_KEY",
            "base_url": "https://openrouter.ai"
        }
    return toml.load(path)
EOF

echo "[*] Writing cost logger..."
cat > codex/cli/logger/__init__.py <<EOF
from pathlib import Path
from datetime import datetime

def log_usage(model, total, prompt, completion):
    log_path = Path("logs/session.log")
    log_path.parent.mkdir(exist_ok=True)
    with log_path.open("a") as f:
        f.write(f"{datetime.now().isoformat()} | {model} | total={total} prompt={prompt} completion={completion}\\n")
EOF

echo "[*] Writing tool calls..."
cat > codex/cli/tools/patcher.py <<EOF
def patch_file(file_path, instruction):
    print(f"[patch] {file_path} <- {instruction}")
EOF

cat > codex/cli/tools/shell_suggester.py <<EOF
def suggest_shell_command(task):
    print(f"[shell] Suggested command for: {task}")
EOF

echo "[*] Writing CLI loop..."
cat > codex/cli/main.py <<EOF
from codex.llm.openrouter import query
from codex.cli.tools import patcher, shell_suggester
from rich.console import Console

console = Console()
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
EOF


echo "[*] Writing Makefile..."
cat > Makefile <<EOF
install:
	$PYTHON_BIN -m venv .venv
	source .venv/bin/activate && pip install requests toml rich

run:
	source .venv/bin/activate && python codex/cli/main.py

clean:
	rm -rf .venv logs/*.log
EOF

# === PATH CHECK ===
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  echo "??  '$INSTALL_DIR' is not in your PATH."
  echo "   You won't be able to run '$CODENAME' globally unless you add it."
  echo "   Add this to your shell config manually if needed:"
  echo "   export PATH=\"$INSTALL_DIR:\$PATH\""
else
  echo "? '$CODENAME' is now globally available. Try running:"
  echo "   $CODENAME --help"
fi