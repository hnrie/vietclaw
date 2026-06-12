# HTTP API reference

Base URL mặc định: `http://127.0.0.1:18636`

- API JSON mount dưới `/api/*`
- Health/status mount ở root: `/health`, `/status`
- Web UI SPA serve tại `/` (fallback static)

Tất cả response JSON dùng `Content-Type: application/json` trừ khi ghi chú khác.

---

## System

### `GET /health`

Health check đơn giản.

**Response 200:**

```json
{ "ok": true }
```

### `GET /status`

Trạng thái daemon.

**Response 200:**

```json
{
  "version": "0.1.0",
  "commit": "dev",
  "uptime": "5m30s",
  "db_ok": true,
  "mode": "eco",
  "max_concurrent_tasks": 1
}
```

### `GET /logs/recent` · `GET /api/logs/recent`

Log gần đây từ file log daemon.

---

## Chat

### `POST /api/chat`

Chat đồng bộ — chờ agent loop hoàn tất rồi trả kết quả.

**Request body:**

| Field | Type | Required | Mô tả |
| --- | --- | --- | --- |
| `message` | string | ✓ | Nội dung user |
| `channel` | string | ✓ | `web`, `discord`, `telegram`, … |
| `user_id` | string | ✓ | User identifier |
| `session_id` | string | | Tạo mới nếu rỗng |
| `agent_id` | string | | Profile ID (mặc định `default`) |
| `mode` | string | | Runtime mode override |

**Response 200:**

```json
{
  "ok": true,
  "session_id": "sess_abc123",
  "agent_id": "default",
  "intent": "chat",
  "reply": "Xin chào!",
  "provider": "mock",
  "model": "mock-small",
  "cost_usd": 0
}
```

**Response 400** (lỗi): `ok: false`, field `error` mô tả lỗi.

### `POST /api/chat/stream`

Chat streaming qua **Server-Sent Events (SSE)**.

**Request body:** giống `/api/chat`.

**Response:** `Content-Type: text/event-stream`

Events:

| Event | Data | Mô tả |
| --- | --- | --- |
| `session` | `{"session_id":"..."}` | Session ID (đầu stream) |
| `text` | `{"text":"..."}` | Token/text chunk từ model |
| `tool_call` | `{"name":"...","input":"..."}` | Agent gọi tool |
| `tool_result` | `{"name":"...","result":"..."}` | Kết quả tool |
| `error` | `{"error":"..."}` | Lỗi — stream kết thúc |
| `done` | `[DONE]` | Hoàn tất |

Ví dụ client (curl):

```sh
curl -N -X POST http://127.0.0.1:18636/api/chat/stream \
  -H 'Content-Type: application/json' \
  -d '{"message":"hi","channel":"web","user_id":"test"}'
```

---

## Sessions

### `GET /api/sessions`

Liệt kê sessions.

**Response 200:** array `Session` objects.

```json
[
  {
    "id": "sess_abc",
    "channel": "web",
    "user_id": "local",
    "title": null,
    "summary": null,
    "created_at": "2026-06-12T10:00:00Z",
    "updated_at": "2026-06-12T10:05:00Z"
  }
]
```

### `GET /api/sessions/{id}`

Chi tiết session + messages.

**Response 200:**

```json
{
  "session": { "id": "...", "channel": "web", ... },
  "messages": [
    { "id": 1, "session_id": "...", "role": "user", "content": "...", "created_at": "..." },
    { "id": 2, "role": "assistant", "content": "...", ... }
  ]
}
```

**Response 404:** session không tồn tại.

---

## Memory

### `GET /api/memory`

Liệt kê memories.

**Query params:**

| Param | Mô tả |
| --- | --- |
| `scope` | Lọc theo scope (thường = `user_id`) |

Limit mặc định: 100 records.

### `POST /api/memory`

Thêm memory thủ công.

**Request body:**

```json
{
  "scope": "local",
  "kind": "note",
  "content": "Server chính chạy port 8080",
  "confidence": "confirmed"
}
```

**Kinds:** `profile`, `preference`, `project`, `workflow`, `decision`, `connection`, `experience`, `note`

**Confidence:** `confirmed` (1.0), `inferred` (0.7), `temporary` (0.35)

**Response 200:** `{ "ok": true, "memory": { ... } }`

### `GET /api/memory/search`

Tìm kiếm hybrid (FTS + vector nếu có embedder).

**Query params:** `scope`, `q`

Limit mặc định: 50.

### `DELETE /api/memory/{id}`

Xóa memory theo ID.

**Response 200:** `{ "ok": true }`

### `POST /api/memory/curate`

Gộp/xóa duplicates trong scope.

**Query params:** `scope`

**Response 200:** `{ "ok": true, "curation": { ... } }`

---

## Settings

### `GET /api/settings`

Trả toàn bộ `config.Config` hiện tại.

### `PUT /api/settings`

Cập nhật config — merge với default, validate, ghi file, hot-reload agent service.

**Request body:** full hoặc partial `config.json` object.

**Response 200:** `{ "ok": true, "config": { ... } }`

### `POST /api/settings/reload`

Đọc lại `config.json` từ disk và apply.

---

## Providers & budget

### `GET /api/providers`

Danh sách providers (API keys **không** expose — chỉ config metadata).

### `GET /api/providers/{id}/models`

Proxy danh sách models từ provider (OpenAI-compatible `/models`).

**Response 200:** `{ "models": [...] }`

### `GET /api/costs/today`

**Response 200:** `{ "total_cost_usd": 0.0123 }`

### `GET /api/budget`

**Response 200:**

```json
{
  "total_cost_usd": 0.01,
  "daily_usd_limit": 0.25,
  "require_approval_above_usd": 0.05,
  "cheap_first": true,
  "allow_escalation": true
}
```

---

## Framework

### `GET /api/framework`

Snapshot framework runtime.

```json
{
  "enabled": true,
  "delegate_enabled": true,
  "hooks_enabled": true,
  "hooks_registered": 0,
  "agents": [...],
  "extensions": [...],
  "channels": [...]
}
```

### `GET /api/framework/extensions`

Built-in extension descriptors.

---

## Channels

### `GET /api/channels`

Trạng thái Discord/Telegram adapters.

### `POST /api/channels/discord/test`

Kiểm tra env token Discord.

**Response:**

```json
{
  "name": "discord",
  "enabled": false,
  "token_env": "VIETCLAW_DISCORD_TOKEN",
  "env_found": true
}
```

### `POST /api/channels/telegram/test`

Tương tự cho Telegram.

---

## Harness

Coding task runner — xem [Harness](harness.md) để hiểu lifecycle.

### `GET /api/harness/runs`

**Query:** `limit` (optional)

**Response:** `{ "runs": [...] }`

### `POST /api/harness/runs`

Tạo harness run.

**Request body:**

```json
{
  "goal": "Fix login bug in auth.go",
  "session_id": "optional",
  "mode": "eco",
  "risk": "medium",
  "workspace_root": "/path/to/repo",
  "auto_run": false,
  "max_tokens": 50000,
  "max_usd": 1.0,
  "max_minutes": 30,
  "allowed_tools": ["file_read", "file_write"],
  "forbidden_tools": ["shell_exec"],
  "success_checks": ["go test ./..."]
}
```

### `GET /api/harness/runs/{id}`

Chi tiết run + events.

### `POST /api/harness/runs/{id}/start`

Bắt đầu thực thi run.

### `POST /api/harness/runs/{id}/cancel`

Hủy run đang chạy.

### `GET /api/harness/runs/{id}/diff`

Trả `text/plain` — git diff cuối cùng.

### `POST /api/harness/runs/{id}/cleanup`

Dọn worktree/branch tạm.

---

## Error format

Lỗi HTTP thường có dạng:

```json
{ "error": "message" }
```

Status codes phổ biến: `400` (bad request), `404` (not found), `405` (method), `500` (internal), `502` (provider upstream).
