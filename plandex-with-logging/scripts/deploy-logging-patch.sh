#!/bin/bash
# deploy-logging-patch.sh
#
# Clone Plandex and apply the LLM logging patch (skeleton package and future modifications).

pwd
exit

set -e

# 1. Clone the official Plandex repository
if [ -d "plandex-with-logging" ]; then
  echo "Directory plandex-with-logging already exists. Remove or rename it before running this script."
  exit 1
fi

git clone https://github.com/plandex-ai/plandex.git plandex-with-logging 

# 2. Copy logging package into correct location
mkdir -p plandex-with-logging/pkg/llmlog
cp -r ./pkg/llmlog/* plandex-with-logging/pkg/llmlog/

# 3. (Future) Copy other modifications as needed
# TODO: Add copy commands for modified files when ready.

echo "LLM logging patch (skeleton) applied. Next steps:"
echo "- Wire up pkg/llmlog into main application."
echo "- Run go build/go test in plandex-with-logging directory."