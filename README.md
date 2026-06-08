# VietClaw

VietClaw is a lightweight personal agent runtime. It is not a model. It is a small Go gateway for coordinating model providers, memory, tools, chat channels, and a minimal web UI.

Phase 1 built the core daemon foundation: local configuration, SQLite storage, logging, health/status endpoints, and a tiny HTML shell.

Phase 2 adds the minimal agent runtime: rule-based intent routing, SQLite memory, provider routing, mock provider, budget checks, context building, tool policy, chat API, and CLI memory/chat commands.

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

## Next Phases

- Real provider presets and approval flow
- Better session summaries and memory curation
- Discord and Telegram channels
- Web UI for chat and memory management
