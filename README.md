# VietClaw

VietClaw is a lightweight personal agent runtime. It is not a model. It is a small Go gateway for coordinating model providers, memory, tools, chat channels, and a minimal web UI.

Phase 1 built the core daemon foundation: local configuration, SQLite storage, logging, health/status endpoints, and a tiny HTML shell.

Phase 2 adds the minimal agent runtime: rule-based intent routing, SQLite memory, provider routing, mock provider, budget checks, context building, tool policy, chat API, and CLI memory/chat commands.

Phase 3 adds chat channel adapters for Discord and Telegram. Channel adapters only normalize inbound messages, apply channel rules, call the existing agent runtime, and send replies back.

## Why Go + SQLite

Go keeps the runtime small, simple to deploy, and friendly to weak VPS machines with 1-2 CPU cores and 1-2GB RAM.

SQLite keeps Phase 1 local-first with no Redis, Postgres, Docker, or external queue service required.

## Run

```sh
go run ./cmd/vietclaw version
go run ./cmd/vietclaw init
go run ./cmd/vietclaw daemon
go run ./cmd/vietclaw status
go run ./cmd/vietclaw doctor
go run ./cmd/vietclaw chat "mày là gì"
go run ./cmd/vietclaw memory add "Minh thích tiết kiệm token"
go run ./cmd/vietclaw memory search "token"
go run ./cmd/vietclaw channels
```

The daemon listens on `127.0.0.1:18636` by default.

## Phase 1 Includes

- CLI commands: `version`, `init`, `daemon`, `status`, `doctor`
- Local config in the VietClaw data directory
- SQLite database initialization
- File and stdout logging
- HTTP endpoints: `/`, `/health`, `/status`, `/logs/recent`
- Embedded minimal web shell

## Phase 2 Includes

- Agent runtime for local chat requests
- Rule-based intent router for `memory_add`, `memory_query`, `chat`, and `action`
- SQLite memory add/list/search
- Provider interface with mock, OpenAI-compatible HTTP, custom HTTP, and optional OpenCode CLI providers
- Context builder with explicit character/history limits
- Budget check from `cost_events`
- Tool policy foundation with shell disabled by default and file tools limited to the workspace
- HTTP APIs: `/api/chat`, `/api/memory`, `/api/memory/search`, `/api/sessions`, `/api/costs/today`, `/api/providers`
- CLI commands: `chat`, `memory list`, `memory add`, `memory search`

## Phase 3 Includes

- Discord adapter using `discordgo`
- Telegram adapter using long polling
- Shared channel policy, mention stripping, session key builder, and in-process idempotency guard
- CLI commands: `channels`, `discord enable`, `discord disable`, `telegram enable`, `telegram disable`
- HTTP APIs: `/api/channels`, `/api/channels/discord/test`, `/api/channels/telegram/test`
- Channel audit tables: `channel_messages`, `channel_events`

## Discord

1. Create a Discord bot in the Discord Developer Portal.
2. Enable Message Content Intent if the bot should read normal guild messages.
3. Set the token in the environment:

```sh
set VIETCLAW_DISCORD_TOKEN=...
go run ./cmd/vietclaw discord enable
go run ./cmd/vietclaw daemon
```

In a Discord guild, VietClaw replies only when mentioned or when a user replies to the bot:

```text
@VietClaw deploy đi
```

In Discord DM, chat normally. No slash commands are registered or required.

## Telegram

1. Create a Telegram bot with BotFather.
2. Set the token in the environment:

```sh
set VIETCLAW_TELEGRAM_TOKEN=...
go run ./cmd/vietclaw telegram enable
go run ./cmd/vietclaw daemon
```

In a Telegram group, VietClaw replies only when mentioned or when a user replies to the bot:

```text
@your_bot_username hỏi gì đó
```

In private chat, chat normally. `/ask` or other command wrappers are not required.

Do not add VietClaw to an untrusted group if dangerous tools are enabled. `shell.exec` is disabled by default.

## Next Phases

- Real provider presets and approval flow
- Better session summaries and memory curation
- Web UI for chat and memory management
