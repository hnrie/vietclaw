# Configuration reference

File cấu hình: `{VIETCLAW_DATA_DIR}/config.json`, tạo bởi `vietclaw init`.

Env vars được load từ `.env` (cwd) và `{data_dir}/.env`. Giá trị đã có trong môi trường OS không bị ghi đè.

## Top-level schema

```json
{
  "server": {},
  "framework": {},
  "runtime": {},
  "database": {},
  "agent": {},
  "channels": {},
  "providers": [],
  "router": {},
  "tools": {},
  "budget": {},
  "agents": []
}
```

Cập nhật runtime qua `PUT /api/settings` hoặc sửa file rồi `POST /api/settings/reload`.

---

## `server`

| Field | Type | Default | Mô tả |
| --- | --- | --- | --- |
| `host` | string | `127.0.0.1` | Bind address |
| `port` | int | `18636` | HTTP port (1–65535) |

---

## `runtime`

| Field | Type | Default | Mô tả |
| --- | --- | --- | --- |
| `mode` | string | `eco` | Runtime mode (`eco`, …) |
| `max_concurrent_tasks` | int | `1` | Số task đồng thời tối đa (≥ 1) |

---

## `database`

| Field | Type | Default | Mô tả |
| --- | --- | --- | --- |
| `path` | string | `{data_dir}/vietclaw.db` | SQLite file path |

---

## `agent`

Cấu hình agent mặc định (profile `default` kế thừa nhiều giá trị này).

| Field | Type | Default | Mô tả |
| --- | --- | --- | --- |
| `experience` | string | `prompt` | `prompt` hoặc `pro` |
| `name` | string | `VietClaw` | Tên hiển thị trong system prompt |
| `language` | string | `vi` | Ngôn ngữ agent (`vi`, `en`) |
| `style` | string | `natural_short` | Phong cách trả lời |
| `default_mode` | string | `eco` | Mode mặc định cho chat request |
| `workspace` | string | `{data_dir}/workspace` | Thư mục file tools |
| `skill_dirs` | []string | `[".codex/skills"]` | Thư mục skills (Codex-style) |
| `max_context_chars` | int | `24000` | Giới hạn ký tự context window |
| `max_history_messages` | int | `12` | Số message lịch sử giữ lại |
| `max_steps` | int | `0` | **0 = unlimited** tool loop steps |
| `max_output_tokens` | int | `0` | **0 = unlimited**; model tự quyết độ dài |
| `reflexion.enabled` | bool | `true` | Lưu lesson khi tool fail |
| `heartbeat.enabled` | bool | `false` | Scheduler chủ động |
| `heartbeat.interval_seconds` | int | `1800` | Chu kỳ heartbeat |
| `heartbeat.session_id` | string | `heartbeat` | Session ID cho heartbeat |
| `heartbeat.user_id` | string | `local` | User ID |
| `heartbeat.prompt` | string | (xem defaults) | Prompt gửi định kỳ |
| `memory_tools.enabled` | bool | `true` | Bật `memory_recall` / `memory_store` |

### Prompt-first defaults

VietClaw ưu tiên UX chat liên tục:

- `max_steps: 0` — không cắt giữa task nhiều bước
- `max_output_tokens: 0` — không cap output giả tạo
- `tools.shell.enabled: false` — shell tắt cho đến khi bạn bật
- `tools.files.workspace_only: true` — file tools chỉ trong workspace

---

## `framework`

| Field | Type | Default | Mô tả |
| --- | --- | --- | --- |
| `enabled` | bool | `true` | Bật extension framework |
| `delegate_enabled` | bool | `true` | Cho phép `agent_delegate` tool |
| `hooks_enabled` | bool | `true` | Lifecycle hooks |

---

## `providers[]`

Mỗi provider:

| Field | Type | Mô tả |
| --- | --- | --- |
| `id` | string | ID duy nhất (dùng trong router, profiles) |
| `type` | string | Xem bảng provider types bên dưới |
| `enabled` | bool | Bật/tắt |
| `default_model` | string | Model mặc định |
| `base_url` | string | Base URL (OpenAI-compatible, custom HTTP) |
| `api_key_env` | string | Tên env var chứa API key |
| `command` | string | Command (opencode-cli) |
| `embed_model` | string | Model embedding (mặc định `text-embedding-3-small`) |
| `cost_per_1k_input` | float | Chi phí USD / 1K input tokens |
| `cost_per_1k_output` | float | Chi phí USD / 1K output tokens |

### Provider types

| `type` | Mô tả |
| --- | --- |
| `mock` | Mock local, không cần key |
| `openai` / `openai-compatible` | OpenAI API hoặc tương thích |
| `anthropic` | Claude API |
| `gemini` | Google Gemini |
| `http` | Custom HTTP adapter |
| `opencode-zen` | OpenCode Zen |
| `opencode-cli` | OpenCode CLI (legacy) |

Nếu không có provider nào `enabled`, hệ thống fallback về mock.

### Ví dụ OpenAI

```json
{
  "id": "openai",
  "type": "openai",
  "enabled": true,
  "default_model": "gpt-4o-mini",
  "api_key_env": "OPENAI_API_KEY",
  "embed_model": "text-embedding-3-small",
  "cost_per_1k_input": 0.00015,
  "cost_per_1k_output": 0.0006
}
```

---

## `router`

| Field | Type | Default | Mô tả |
| --- | --- | --- | --- |
| `default_provider` | string | `mock` | Provider mặc định |
| `default_model` | string | `mock-small` | Model mặc định |
| `intent_mode` | string | `hybrid` | Cách phân loại intent |
| `agent_routing` | string | `hybrid` | Routing agent/provider |
| `cheap_first` | bool | `true` | Ưu tiên model rẻ trước |
| `allow_escalation` | bool | `true` | Cho phép escalate sang model mạnh hơn |

---

## `tools`

### `tools.files`

| Field | Type | Default | Mô tả |
| --- | --- | --- | --- |
| `enabled` | bool | `true` | File read/write/list/grep/find |
| `workspace_only` | bool | `true` | Chặn path ngoài workspace |

### `tools.shell`

| Field | Type | Default | Mô tả |
| --- | --- | --- | --- |
| `enabled` | bool | `false` | **Tắt mặc định** |
| `sandbox` | string | `none` | `none` hoặc `docker` |
| `docker_binary` | string | `docker` | Path docker binary |
| `docker_image` | string | `alpine:3.20` | Image khi sandbox=docker |
| `docker_network` | string | `none` | Docker network mode |
| `workspace_mode` | string | `ro` | `ro` hoặc `rw` mount workspace |
| `timeout_seconds` | int | `30` | Timeout mỗi lệnh |

### `tools.shell.network_policy`

| Field | Type | Default | Mô tả |
| --- | --- | --- | --- |
| `enabled` | bool | `true` | Bật network policy |
| `restrict_to_allow_hosts` | bool | `false` | Chỉ cho phép hosts trong allowlist |
| `allow_hosts` | []string | `[]` | Host được phép |
| `deny_hosts` | []string | metadata endpoints | Host bị chặn |
| `deny_private` | bool | `true` | Chặn IP private/metadata |

### `tools.mcp[]`

MCP server động — tools được discover lúc khởi động registry.

| Field | Type | Mô tả |
| --- | --- | --- |
| `id` | string | ID server |
| `enabled` | bool | Bật/tắt |
| `transport` | string | Loại transport |
| `url` | string | URL (HTTP/SSE transport) |
| `command` | string | Command spawn (stdio) |
| `args` | []string | Args |
| `env` | map | Env vars |
| `timeout_seconds` | int | Timeout |
| `install_command` | string | Auto-install nếu thiếu |
| `install_args` | []string | Args install |

---

## `budget`

| Field | Type | Default | Mô tả |
| --- | --- | --- | --- |
| `daily_usd_limit` | float | `0.25` | Giới hạn chi phí/ngày (USD) |
| `require_approval_above_usd` | float | `0.05` | Yêu cầu approval trên ngưỡng |

Chi phí được ghi vào bảng `cost_events` và tra qua `GET /api/budget`.

---

## `channels`

### Discord

| Field | Type | Default | Mô tả |
| --- | --- | --- | --- |
| `enabled` | bool | `false` | Bật Discord bot |
| `token_env` | string | `VIETCLAW_DISCORD_TOKEN` | Env var token |
| `allowed_guilds` | []string | `[]` | Rỗng = tất cả guilds |
| `allowed_channels` | []string | `[]` | Rỗng = tất cả channels |
| `respond_in_guilds` | string | `mention_or_reply` | `always`, `never`, `mention_or_reply` |
| `respond_in_dm` | bool | `true` | Phản hồi DM |

### Telegram

| Field | Type | Default | Mô tả |
| --- | --- | --- | --- |
| `enabled` | bool | `false` | Bật Telegram bot |
| `token_env` | string | `VIETCLAW_TELEGRAM_TOKEN` | Env var token |
| `allowed_chats` | []string | `[]` | Rỗng = tất cả chats |
| `respond_in_groups` | string | `mention_or_reply` | Policy trong group |
| `respond_in_private` | bool | `true` | Phản hồi chat riêng |
| `poll_timeout_seconds` | int | `30` | Long-poll timeout |

### Attachments

| Field | Type | Default | Mô tả |
| --- | --- | --- | --- |
| `enabled` | bool | `true` | Nhận file đính kèm |
| `max_files` | int | `5` | Số file tối đa / message |
| `max_bytes` | int | `524288` | Kích thước tối đa / file (512 KB) |
| `allowed_extensions` | []string | text extensions | Extension cho phép |

---

## `agents[]` — Agent profiles

| Field | Type | Mô tả |
| --- | --- | --- |
| `id` | string | ID duy nhất (`default` là profile mặc định) |
| `name` | string | Tên hiển thị |
| `language` | string | Override ngôn ngữ |
| `persona` | string | System persona bổ sung |
| `tools` | []string | Allowlist tools (rỗng = tất cả theo policy) |
| `providers` | []string | Allowlist provider IDs (rỗng = router default) |
| `memory_scope` | string | Scope memory riêng (rỗng = theo user_id) |
| `max_steps` | int | Override `agent.max_steps` cho profile này |

### Ví dụ researcher profile

```json
{
  "agents": [
    {
      "id": "researcher",
      "name": "Researcher",
      "persona": "Focus on research. Delegate coding to other agents.",
      "tools": ["web_search", "web_fetch", "memory_recall", "agent_delegate"],
      "providers": ["zen"],
      "memory_scope": "team-research"
    },
    {
      "id": "coder",
      "name": "Coder",
      "persona": "Write and verify code in the workspace.",
      "tools": ["file_read", "file_write", "file_grep", "shell_exec"],
      "providers": ["openai"]
    }
  ]
}
```

---

## Environment variables

| Variable | Mục đích |
| --- | --- |
| `VIETCLAW_DATA_DIR` | Override data directory |
| `OPENAI_API_KEY` | OpenAI / compatible |
| `GEMINI_API_KEY` | Google Gemini |
| `ANTHROPIC_API_KEY` | Anthropic |
| `OPENCODE_ZEN_KEY` | OpenCode Zen |
| `VIETCLAW_DISCORD_TOKEN` | Discord bot |
| `VIETCLAW_TELEGRAM_TOKEN` | Telegram bot |

---

## Validation rules

`config.Validate()` kiểm tra khi save/reload:

- `server.port` trong 1–65535
- `runtime.max_concurrent_tasks` ≥ 1
- `agent.max_steps`, `agent.max_output_tokens` ≥ 0
- `budget.*` ≥ 0
- `tools.shell.sandbox` ∈ `{none, docker}`
- `tools.shell.workspace_mode` ∈ `{ro, rw}`
- Mỗi provider và agent profile phải có `id` và `type`/`id` hợp lệ
