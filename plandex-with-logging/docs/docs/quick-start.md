---
sidebar_position: 2
sidebar_label: Quickstart
---

# Quickstart

## Install Plandex

```bash
curl -sL https://plandex.ai/install.sh | bash
```

[Click here for more installation options.](./install.md)

Note that Windows is supported via [WSL](https://learn.microsoft.com/en-us/windows/wsl/about). Plandex only works correctly on Windows in the WSL shell. It doesn't work in the Windows CMD prompt or PowerShell.

## Hosting Options

| Option                                | Description                                                                                                                                                                                                                                                 |
| ------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Plandex Cloud (Integrated Models)** | • No separate accounts or API keys.<br/>• Easy multi-device usage.<br/>• Centralized billing, budgeting, usage tracking, and cost reporting.<br/>• Quickest way to [get started.](https://app.plandex.ai/start?modelsMode=integrated)                                                        |
| **Plandex Cloud (BYO API Key)**       | • Use Plandex Cloud with your own [OpenRouter.ai](https://openrouter.ai) key (or [other model provider](./models/model-providers.md) accounts and API keys).<br/>• [Get started](https://app.plandex.ai/start?modelsMode=byo)                                                                   |
| **Self-hosted/Local Mode**            | • Run Plandex locally with Docker or host on your own server.<br/>• Use your own [OpenRouter.ai](https://openrouter.ai) key (or [other model provider](./models/model-providers.md) accounts and API keys).<br/>• Follow the [local-mode quickstart](./hosting/self-hosting/local-mode-quickstart.md) to get started. |

If you're going with a 'BYO API Key' option above (whether cloud or self-hosted), decide on the model provider(s) you want to use. The quickest option is to use OpenRouter.ai, but you can also use [many other providers](./models/model-providers.md).

If you're using OpenRouter.ai, first [sign up here.](https://openrouter.ai/signup) Then [generate an API key here.](https://openrouter.ai/keys) Set the `OPENROUTER_API_KEY` environment variable:

```bash
export OPENROUTER_API_KEY=...
```

## Get Started

If you're starting on a new project, make a directory first:

```bash
mkdir your-project-dir
```

Now `cd` into your **project's directory.**

```bash
cd your-project-dir
```

Then just give a quick the REPL help text a quick read, and you're ready go. The REPL starts in _chat mode_ by default, which is good for fleshing out ideas before moving to implementation. Once the task is clear, Plandex will prompt you to switch to _tell mode_ to make a detailed plan and start writing code.

```bash
plandex
```

or for short:

```bash
pdx
```

☁️ _If you're using Plandex Cloud, you'll be prompted at this point to start a trial._

Then just give the REPL help text a quick read, and you're ready go.
