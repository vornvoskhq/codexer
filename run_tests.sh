#!/usr/bin/env bash
set -e

if [ ! -d ".venv" ]; then
    echo "Creating virtual environment in .venv"
    python3 -m venv .venv
fi

# shellcheck disable=SC1091
source .venv/bin/activate

echo "Installing/updating dependencies from requirements.txt..."
#pip install --upgrade pip
#pip install -r requirements.txt

echo "Running pytest..."
PYTHONPATH=$(pwd) pytest "$@"
exit $?