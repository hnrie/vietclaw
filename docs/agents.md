# Agents & framework

VietClaw không chỉ là một chatbot — có agent loop đa bước, profiles, delegation, hooks, và routing provider.

## Agent loop

Luồng chính (`internal/agent/loop.go`):

```
ChatRequest
    → normalize + apply profile
    → ensure session + save user message
    → classify intent (router)
    → insert agent_run (status: running)
    → [memory_add|memory_query] → rule handler (fast path)
    → [chat|action] → agentic loop:
          build context (system + memories + history + skills)
          → LLM chat (with tools)
          → tool calls? execute → append results → loop
          → finish run (completed/failed/blocked/needs_approval)
    → save assistant message + return reply
```

### Intent classification

Rule-based (`internal/router/intent.go`) — không cần LLM:

| Intent | Trigger examples |
| --- | --- |
| `memory_add` | "nhớ là", "lưu lại", "remember that" |
| `memory_query` | "nhớ gì", "what do you remember", "recall" |
| `action` | "chạy ", "đọc file", "run ", "read file" |
| `chat` | Mặc định |

`memory_add` / `memory_query` bypass full agentic loop — phản hồi nhanh qua rule handlers.

### Tool loop limits

| Config | Ý nghĩa |
| --- | --- |
| `agent.max_steps: 0` | Unlimited steps (default) |
| `agent.max_steps: N` | Dừng sau N vòng LLM↔tool |
| `agents[].max_steps` | Override per profile |

Khi đạt max steps, agent finalize với text đã tích lũy + thông báo.

### Run statuses

| Status | Mô tả |
| --- | --- |
| `running` | Đang thực thi |
| `completed` | Thành công |
| `failed` | Lỗi (provider, tool, context) |
| `blocked` | Bị chặn policy |
| `needs_approval` | Vượt budget approval threshold |

Runs lưu trong `agent_runs` với `parent_run_id` cho delegation tree.

---

## Context building

`internal/context/builder.go` assemble messages gửi LLM:

1. **System prompt** — base + agent name (i18n theo `agent.language`)
2. **Relevant memories** — hybrid search top 9, split core (6) + experiences/lessons (3)
3. **Skills** — load từ `agent.skill_dirs` (Codex-style skill files)
4. **Session history** — trim theo `max_history_messages` và `max_context_chars`
5. **Persona** — từ agent profile nếu có

Memory scope mặc định = `user_id` từ chat request, hoặc `agents[].memory_scope` nếu set.

---

## Agent profiles

Định nghĩa trong `config.agents[]`. Profile `default` luôn tồn tại sau `init`.

| Field | Tác dụng |
| --- | --- |
| `persona` | Inject vào system context |
| `tools` | Allowlist — rỗng = all tools theo global policy |
| `providers` | Allowlist provider IDs cho loop |
| `language` | Override i18n tool descriptions + prompts |
| `memory_scope` | Isolate memory giữa agents |

Chọn profile qua `agent_id` trong chat request hoặc channel routing.

### Ví dụ delegation setup

```json
{
  "framework": {
    "enabled": true,
    "delegate_enabled": true
  },
  "agents": [
    {
      "id": "default",
      "name": "Orchestrator",
      "tools": ["agent_delegate", "memory_recall", "web_search"]
    },
    {
      "id": "coder",
      "name": "Coder",
      "persona": "You write Go code only.",
      "tools": ["file_read", "file_write", "file_grep"],
      "providers": ["openai"]
    }
  ]
}
```

Parent agent gọi `agent_delegate(agent_id="coder", message="...")` → child run với tracing.

---

## Provider routing

`internal/router` chọn provider/model mỗi vòng loop:

- `router.cheap_first` — thử model rẻ trước
- `router.allow_escalation` — escalate khi cần
- Profile `providers[]` — giới hạn pool
- Fallback chain khi provider error
- Cost tracking → `cost_events` + budget checks

Embedder cho memory search: `SelectDefaultEmbedder()` từ provider có `embed_model`.

---

## Framework hooks

Lifecycle events (`internal/framework/hooks.go`):

| Event | Khi nào |
| --- | --- |
| `before_chat` | Trước khi xử lý message |
| `after_chat` | Sau khi có reply |
| `before_tool` | Trước execute tool |
| `after_tool` | Sau execute tool |
| `run_start` | Agent run bắt đầu |
| `run_finish` | Agent run kết thúc |

`HookContext` chứa: `session_id`, `agent_id`, `run_id`, `parent_run_id`, message/reply, tool name/input/output/error, metadata.

Hooks đăng ký programmatic (extensions/plugins) — `GET /api/framework` trả `hooks_registered` count.

Bật/tắt: `framework.hooks_enabled`.

---

## Reflexion

Khi tool fail (`agent.reflexion.enabled`):

- Tự động lưu verbal lesson vào memory kind `experience`
- Lessons xuất hiện trong context với prefix `[lesson]`
- Inspired by Reflexion paper (Shinn et al., 2023)

---

## Heartbeat (proactive)

`agent.heartbeat` — scheduler gửi prompt định kỳ vào session riêng:

```json
{
  "heartbeat": {
    "enabled": true,
    "interval_seconds": 1800,
    "session_id": "heartbeat",
    "user_id": "local",
    "prompt": "Heartbeat: review pending tasks..."
  }
}
```

Dùng cho reminders, proactive checks — tắt mặc định.

---

## Experience modes

| `agent.experience` | Mô tả |
| --- | --- |
| `prompt` | Prompt-first, unlimited steps (default) |
| `pro` | Pro mode với constraints khác (xem config/UI) |

---

## Inspect runtime

```sh
curl -s http://127.0.0.1:18636/api/framework | jq
```

Trả: framework flags, agent profiles, registered channel adapters, builtin extensions.
