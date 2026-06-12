# Architecture

Tổng quan kiến trúc runtime VietClaw — single Go binary, SQLite, embedded Nuxt SPA.

---

## High-level diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     vietclaw daemon                          │
│  ┌──────────┐  ┌─────────────┐  ┌────────────────────────┐ │
│  │ HTTP     │  │ Channels    │  │ Agent Service          │ │
│  │ Router   │──│ Manager     │──│  loop · profiles       │ │
│  │ + Static │  │ Discord     │  │  memory · tools        │ │
│  │   UI     │  │ Telegram    │  │  router · context      │ │
│  └──────────┘  └─────────────┘  └───────────┬────────────┘ │
│                                               │              │
│  ┌────────────────────────────────────────────┴────────────┐ │
│  │ SQLite (vietclaw.db)                                     │ │
│  │ sessions · messages · memories · agent_runs · costs    │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
         ▲                              ▲
         │ go:embed                     │ HTTPS (optional)
   internal/web/dist              LLM provider APIs
   (Nuxt build output)
```

---

## Module map

| Package | Vai trò |
| --- | --- |
| `cmd/vietclaw` | CLI: init, setup, daemon, doctor, status |
| `internal/app` | Application wiring — config, DB, agent, channels |
| `internal/web` | HTTP handlers, SSE streaming, static SPA |
| `internal/agent` | Chat loop, sessions, runs, delegation, reflexion |
| `internal/tools` | Tool registry, policy, MCP discovery |
| `internal/memory` | Store, FTS, vector hybrid search |
| `internal/providers` | LLM adapters (OpenAI, Anthropic, Gemini, mock, …) |
| `internal/router` | Intent classify, model routing, cost tracking |
| `internal/context` | Message/context builder, history trim |
| `internal/channels` | Discord, Telegram adapters, policy |
| `internal/framework` | Hooks registry, extensions |
| `internal/harness` | Coding task runner |
| `internal/config` | Load/save/validate/merge config |
| `internal/db` | Schema, migrations, health |
| `internal/skills` | Skill file loader |
| `internal/i18n` | vi/en catalogs |
| `apps/web` | Nuxt 4 source (build → embed) |

---

## Startup sequence (`daemon`)

1. Load `config.json` + env
2. Open SQLite, run migrations
3. Init `agent.Service` + `framework.Framework`
4. Build tool registry (discover MCP)
5. Start channel manager (if enabled)
6. Listen HTTP `server.host:server.port`
7. Optional: heartbeat scheduler

---

## Request flow — web chat

```
POST /api/chat/stream
  → web.handleAPIChatStream
  → agent.Service.ChatStream
       → router.Classify(intent)
       → insert agent_run
       → StreamAgenticLoop
            → context.Builder.Messages()
            → providers.Chat (streaming)
            → tools.Execute (per tool_call)
            → framework hooks emit
            → finish run
  → SSE events to client
```

---

## Database schema

Core tables (`internal/db/schema.go`):

| Table | Mục đích |
| --- | --- |
| `settings` | Key-value settings |
| `events` | Generic event log |
| `memories` | Long-term memory + embeddings |
| `sessions` | Chat sessions |
| `messages` | Session messages |
| `cost_events` | Token/cost tracking |
| `providers` | Provider state (if persisted) |
| `tool_events` | Tool call audit log |
| `agent_runs` | Agent run tracing + delegation tree |
| `harness_runs` | Harness task state |
| `harness_events` | Harness step events |
| `channel_messages` | Idempotency + channel audit |
| `channel_events` | Channel-level events |

Single file SQLite — backup = copy `vietclaw.db`.

---

## Web UI embed

```
apps/web  ──pnpm build──►  .output/public
                │
                └──copy-dist.mjs──►  internal/web/dist/
                                           │
                              go build (//go:embed)
                                           │
                                    vietclaw binary
```

`internal/web/static_handler.go` serve embedded FS, SPA fallback `index.html`.

Dev mode: `pnpm dev` proxies API tới daemon — không cần rebuild binary.

---

## Extension points

| Mechanism | Mô tả |
| --- | --- |
| Agent profiles | Config-driven personas + tool/provider allowlists |
| `agent_delegate` | Multi-agent orchestration |
| Framework hooks | `before_chat`, `after_tool`, … |
| MCP servers | Dynamic tool registration |
| `internal/plugins` | Builtin extension registry |
| Channel registry | Adapter pattern cho platforms mới |

---

## Concurrency & runtime

- `runtime.max_concurrent_tasks` — giới hạn task song song
- `runtime.mode` — `eco` default, ảnh hưởng provider selection
- Context builder dùng `singleflight` cho summarize dedup
- Channel adapters chạy goroutines riêng (long-poll Telegram, Discord gateway)

---

## i18n

Song ngữ `vi` (default) / `en`:

- CLI messages: `internal/i18n`
- Tool descriptions: theo `agent.language` hoặc profile language
- Web UI: `apps/web/locales/{vi,en}.json`

---

## Security layers

```
Layer 1: Config defaults (shell off, workspace_only on)
Layer 2: Tool Policy (path resolve, sandbox)
Layer 3: Shell network_policy (deny private/metadata)
Layer 4: Agent profile tool allowlist
Layer 5: Harness per-run allow/forbid + budget
Layer 6: Budget approval threshold
```

---

## Testing

```
tests/           — integration tests (channels ~10s)
go test ./...    — unit + integration
CI               — pnpm build web → go test → go vet
```

---

## Related docs

- [Configuration](configuration.md)
- [HTTP API](api.md)
- [Agents & framework](agents.md)
- [Tools](tools.md)
