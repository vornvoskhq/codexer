install:
	/home/vorn/.pyenv/shims/python3 -m venv .venv
	source .venv/bin/activate && pip install requests toml rich

run:
	source .venv/bin/activate && python codex/cli/main.py

clean:
	rm -rf .venv logs/*.log
