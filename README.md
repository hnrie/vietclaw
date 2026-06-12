# VietClaw

[![CI](https://github.com/vietclaw/vietclaw/actions/workflows/ci.yml/badge.svg)](https://github.com/vietclaw/vietclaw/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev/)

**VietClaw** is a lightweight, local-first **agent framework** for personal automation. Chat in natural language (Vietnamese or English); the agent handles memory, tools, delegation, and channels. No complex setup required.

Single Go binary · SQLite · embedded web UI · Discord & Telegram · built-in coding harness.

---

## Documentation

In-depth guides live in **[docs/](docs/README.md)** — configuration reference, HTTP API, tools, agents, memory, channels, harness, and architecture.

## Table of contents

- [Why VietClaw](#why-vietclaw)
- [Features](#features)
- [Quick start](#quick-start)
- [Installation](#installation)
- [Configuration](#configuration)
- [CLI](#cli)
- [Channels](#channels)
- [Web UI](#web-ui)
- [Agent framework](#agent-framework)
- [Development](#development)
- [Security](#security)
- [Project structure](#project-structure)
- [Contributing](#contributing)
- [License](#license)

## Why VietClaw

| | VietClaw | Typical cloud agent stacks |
| --- | --- | --- |
| Deploy | One binary, no Node at runtime | Multiple services, containers |
| Data | SQLite on disk, your machine | Remote DB, vendor lock-in |
| RAM | Tuned for small VPS (1–2 GB) | Often heavier |
| Defaults | `shell_exec` off, workspace-scoped files | Often requires manual hardening |

Go keeps the daemon small and easy to ship. SQLite keeps everything local without Redis, Postgres, or Docker.

## Features

- **Prompt-first chat** — memory, web search, files, shell (when enabled), sub-agent delegation
- **Multi-provider routing** — OpenAI-compatible APIs, Gemini, Anthropic, OpenCode Zen, mock
- **Long-term memory** — hybrid FTS + vector recall; active `memory_recall` / `memory_store` tools
- **Agent framework** — profiles, hooks, tool registry, `agent_delegate` with run tracing
- **Channels** — Discord and Telegram (mention/reply gating in groups)
- **Web UI** — Nuxt chat workspace + advanced console (memory, providers, budget, logs)
- **Harness** — plan / verify / worktree runner for coding tasks
- **Research-inspired runtime** — tiered context, reflexion on tool failures, optional heartbeat scheduler

## Quick start

**Requirements:** Go 1.25+

```sh
git clone https://github.com/vietclaw/vietclaw.git
cd vietclaw

go run ./cmd/vietclaw init
go run ./cmd/vietclaw daemon
```

Open **http://127.0.0.1:18636** — chat immediately (mock provider works out of the box).

Add a real model provider via environment variable and config (see [Configuration](#configuration)), or use the web UI **Công cụ nâng cao** drawer.

```sh
go run ./cmd/vietclaw setup
go run ./cmd/vietclaw status
go run ./cmd/vietclaw doctor
```

## Installation

### From source (recommended for development)

```sh
# Build embedded web UI (required before go build)
cd apps/web && pnpm install && pnpm build && cd ../../

go build -o vietclaw ./cmd/vietclaw
./vietclaw init
./vietclaw daemon
```

The release workflow produces binaries for Linux, Windows, and macOS (amd64/arm64) on tagged releases (`v*`).

### Data directory

| OS | Default path |
| --- | --- |
| All platforms | `~/.vietclaw/` (Windows: `%USERPROFILE%\.vietclaw\`) |

Override with `VIETCLAW_DATA_DIR`. Contains `config.json`, `vietclaw.db`, `workspace/`, and `logs/`.

On Windows, older installs used `%APPDATA%\VietClaw\` — run `vietclaw doctor` for a migration hint if config is missing.

## Configuration

Config file: `config.json` in the data directory. Created by `vietclaw init`.

**Environment variables** (also loaded from `.env` in cwd or data dir):

| Variable | Purpose |
| --- | --- |
| `GEMINI_API_KEY` | Google Gemini provider |
| `OPENAI_API_KEY` | OpenAI-compatible providers |
| `ANTHROPIC_API_KEY` | Anthropic provider |
| `OPENCODE_ZEN_KEY` | OpenCode Zen provider |
| `VIETCLAW_DISCORD_TOKEN` | Discord bot |
| `VIETCLAW_TELEGRAM_TOKEN` | Telegram bot |

**Sensible defaults (prompt-first):**

- `agent.max_steps: 0` — unlimited tool loops (no mid-task cutoff)
- `agent.max_output_tokens: 0` — no artificial output cap; model decides length
- `tools.shell.enabled: false` — shell execution off until you enable it
- `tools.files.workspace_only: true` — file tools scoped to workspace

Example agent profile:

```json
{
  "agents": [
    {
      "id": "researcher",
      "name": "Researcher",
      "persona": "Focus on research. Delegate coding to other agents.",
      "tools": ["web_search", "web_fetch", "memory_recall", "agent_delegate"],
      "providers": ["zen"]
    }
  ],
  "framework": {
    "enabled": true,
    "delegate_enabled": true,
    "hooks_enabled": true
  }
}
```

## CLI

| Command | Description |
| --- | --- |
| `vietclaw version` | Print version and commit |
| `vietclaw init` | Create data dir, config, database |
| `vietclaw setup` | Interactive first-time configuration |
| `vietclaw daemon` | Start HTTP server and channels |
| `vietclaw status` | Query running daemon |
| `vietclaw doctor` | Health checks |

Chat, memory, providers, and channels are configured via `setup` or the web UI — not separate CLI commands.

## Channels

### Discord

1. Create a bot in the [Discord Developer Portal](https://discord.com/developers/applications).
2. Enable **Message Content Intent** for guild messages.
3. Set `VIETCLAW_DISCORD_TOKEN` and enable:

```sh
vietclaw setup   # enable Discord in the wizard, or edit config.json
vietclaw daemon
```

In guilds, the bot responds on **mention or reply**. In DM, chat normally.

### Telegram

1. Create a bot via [BotFather](https://t.me/Botfather).
2. Set `VIETCLAW_TELEGRAM_TOKEN` and enable:

```sh
vietclaw setup   # enable Telegram in the wizard, or edit config.json
vietclaw daemon
```

In groups, respond on **mention or reply**. Do not add the bot to untrusted groups if dangerous tools are enabled.

## Web UI

The daemon serves an embedded SPA at `/`. Source lives in `apps/web` (Nuxt 4 + Tailwind v4).

**End users** only need the binary — UI is embedded at build time.

**Developers** — proxy to Vite while iterating:

```sh
go run ./cmd/vietclaw daemon          # backend
cd apps/web && pnpm dev               # frontend on another port
```

Production embed:

```sh
cd apps/web && pnpm build             # copies to internal/web/dist
go build ./cmd/vietclaw
```

`internal/web/dist` is **not** committed; CI builds it before `go test` / release.

## Agent framework

| Capability | Description |
| --- | --- |
| Agent profiles | Persona, tools, providers, memory scope, language |
| `agent_delegate` | Spawn child runs with `parent_run_id` tracing |
| Lifecycle hooks | `before_chat`, `after_chat`, `before_tool`, `after_tool`, `run_start`, `run_finish` |
| Registries | Tools, channel adapters; inspect via CLI or `/api/framework` |

Optional proactive **heartbeat** (config `agent.heartbeat`) and **reflexion** on tool failures (`agent.reflexion`).

## Development

```sh
# Backend tests (CI also builds web first)
cd apps/web && pnpm install && pnpm build && cd ../..
go test ./...

# Web only
cd apps/web && pnpm typecheck && pnpm build
```

See [CONTRIBUTING.md](.github/CONTRIBUTING.md) for PR workflow, code style, and issue templates.

## Security

- Shell execution disabled by default; enable only on trusted machines.
- File tools restricted to workspace unless configured otherwise.
- Shell network policy denies private/metadata endpoints by default.
- Report vulnerabilities per [SECURITY.md](.github/SECURITY.md) — **do not** open public issues for security bugs.

## Project structure

```
docs/                  In-depth documentation
cmd/vietclaw/          CLI entrypoint
internal/agent/        Agent loop, profiles, delegation
internal/tools/        Tool registry and implementations
internal/memory/       SQLite memory store
internal/providers/    LLM provider adapters
internal/channels/     Discord, Telegram
internal/web/          HTTP handlers + embedded UI
internal/framework/    Hooks and extension framework
apps/web/              Nuxt web UI source
tests/                 Integration tests
```

## Contributing

We welcome issues and pull requests. Please read:

- [Contributing guide](.github/CONTRIBUTING.md)
- [Code of conduct](.github/CODE_OF_CONDUCT.md)

## License

[MIT License](LICENSE) — Copyright 2026 Lê Hùng Quang Minh
