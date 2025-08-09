import os
import requests

def query(messages, api_key=None, model='glm-4.5-air', base_url=None, stream=False):
    """
    Send messages to the OpenRouter API and return the response content (or stream).  
    messages: list of {role, content} dicts.  
    """
    key = api_key or os.getenv('OPENROUTER_API_KEY')
    if not key:
        raise ValueError('OPENROUTER_API_KEY is not set')
    url = f"{base_url or 'https://openrouter.ai'}/api/v1/chat/completions"
    headers = {
        'Authorization': f'Bearer {key}',
        'Content-Type': 'application/json',
    }
    payload = {
        'model': model,
        'messages': messages,
        'stream': stream,
    }
    resp = requests.post(url, headers=headers, json=payload, stream=stream, timeout=60)
    resp.raise_for_status()
    if stream:
        for chunk in resp.iter_lines(decode_unicode=True):
            if chunk:
                yield chunk
    else:
        data = resp.json()
        return data['choices'][0]['message']['content']