from pathlib import Path
from datetime import datetime

def log_usage(model, total, prompt, completion):
    log_path = Path("logs/session.log")
    log_path.parent.mkdir(exist_ok=True)
    with log_path.open("a") as f:
        f.write(f"{datetime.now().isoformat()} | {model} | total={total} prompt={prompt} completion={completion}\n")
