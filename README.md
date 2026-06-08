# VietClaw

VietClaw is a lightweight personal agent runtime. It is not a model. It is a small Go gateway for coordinating model providers, memory, tools, chat channels, and a minimal web UI.

Phase 1 focuses on the core daemon foundation: local configuration, SQLite storage, logging, health/status endpoints, and a tiny HTML shell.

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
```

The daemon listens on `127.0.0.1:18636` by default.

## Phase 1 Includes

- CLI commands: `version`, `init`, `daemon`, `status`, `doctor`
- Local config in the VietClaw data directory
- SQLite database initialization
- File and stdout logging
- HTTP endpoints: `/`, `/health`, `/status`, `/logs/recent`
- Embedded minimal web shell

## Next Phases

- Agent runtime loop
- Memory APIs and retrieval
- Provider router
- Discord and Telegram channels
- Budget-aware task execution

