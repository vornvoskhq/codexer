print("[DEBUG] config.py loaded from:", __file__)
import os
import toml

def load_config():
    config_path = os.path.expanduser("~/.codex/config.toml")
    cfg = toml.load(config_path)

    print("[DEBUG] (__init__.py) Loaded config keys:", list(cfg.keys()))

   # Fallback if system_prompt is still missing
    if "system_prompt" not in cfg:
        print("[WARN] system_prompt not found in config. Using default.")
        cfg["system_prompt"] = "You are Niggex, a terse software engineer."

    # Load system prompt from file if specified
    if "system_prompt_file" in cfg:
        prompt_path = os.path.expanduser(cfg["system_prompt_file"])
        print("[DEBUG] Attempting to load system prompt from:", prompt_path)
        try:
            with open(prompt_path, "r") as f:
                cfg["system_prompt"] = f.read()
            print("[DEBUG] Loaded system prompt successfully.")
        except Exception as e:
            print(f"[ERROR] Failed to load system_prompt_file: {e}")
            cfg["system_prompt"] = "You are Riggex, a salty sea captain."

 
    return cfg

