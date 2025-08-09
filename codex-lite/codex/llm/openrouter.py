"""
import requests, json
from codex.cli.config import load_config
from codex.cli.logger import log_usage

def query(prompt):
    cfg = load_config()
    print(cfg)
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

    if not res.ok:
        raise RuntimeError(f"Request failed: {res.status_code} {res.reason}\n{res.text}")

    try:
        data = res.json()
    except Exception as e:
        raise RuntimeError(f"Failed to parse JSON response: {e}\nRaw response:\n{res.text}")

    if "choices" not in data:
        raise RuntimeError(f"Unexpected response format:\n{json.dumps(data, indent=2)}")

    usage = data.get("usage", {})
    log_usage(cfg['model'], usage.get("total_tokens", 0), usage.get("prompt_tokens", 0), usage.get("completion_tokens", 0))
    return data["choices"][0]["message"]["content"]
"""    
    
    
import requests, json
from codex.cli.config import load_config
from codex.cli.logger import log_usage

_cfg = load_config()

def _post(messages):
    headers = {
        "Authorization": f"Bearer {_cfg['api_key']}",
        "Content-Type": "application/json"
    }
    payload = {
        "model": _cfg['model'],
        "messages": messages
    }
    url = f"{_cfg.get('base_url', 'https://openrouter.ai')}/api/v1/chat/completions"
    res = requests.post(url, headers=headers, data=json.dumps(payload))

    if not res.ok:
        raise RuntimeError(f"Request failed: {res.status_code} {res.reason}\n{res.text}")

    try:
        data = res.json()
    except Exception as e:
        raise RuntimeError(f"Failed to parse JSON response: {e}\nRaw response:\n{res.text}")

    if "choices" not in data:
        raise RuntimeError(f"Unexpected response format:\n{json.dumps(data, indent=2)}")

    usage = data.get("usage", {})
    log_usage(_cfg['model'], usage.get("total_tokens", 0), usage.get("prompt_tokens", 0), usage.get("completion_tokens", 0))
    print(data)
    return data["choices"][0]["message"]["content"]

def send_initial_prompt():
    try:
        with open(_cfg["system_prompt_file"], "r") as f:
            long_prompt = f.read()
        print("[DEBUG] Loaded system prompt successfully.")
    except Exception as e:
        raise RuntimeError(f"Failed to load system prompt file: {e}")

    return _post([
        {"role": "system", "content": long_prompt},
        {"role": "user", "content": "hi"}  # or any session-start trigger
    ])

def query(prompt):
    return _post([
        {"role": "system", "content": _cfg["system_prompt"]},
        {"role": "user", "content": prompt}
    ])