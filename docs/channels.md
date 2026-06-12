# Channels

VietClaw bridge chat từ Discord và Telegram vào cùng agent loop với web UI.

## Kiến trúc

```
Discord/Telegram event
    → channels.Manager
    → policy check (mention/reply/DM)
    → session mapping (channel_messages)
    → agent.Chat / ChatStream
    → reply outbound
```

Channel adapters đăng ký trong `internal/channels/registry.go`. Status qua `GET /api/channels`.

---

## Discord

### Setup

1. [Discord Developer Portal](https://discord.com/developers/applications) → tạo Application → Bot
2. Bật **Message Content Intent** (cần cho đọc nội dung guild)
3. Invite bot với permissions: Send Messages, Read Message History
4. Set env: `VIETCLAW_DISCORD_TOKEN=...`
5. Enable trong config:

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token_env": "VIETCLAW_DISCORD_TOKEN",
      "respond_in_guilds": "mention_or_reply",
      "respond_in_dm": true
    }
  }
}
```

### Policy

| Setting | Values | Default |
| --- | --- | --- |
| `respond_in_guilds` | `mention_or_reply`, `always`, `never` | `mention_or_reply` |
| `respond_in_dm` | bool | `true` |

`mention_or_reply`: trong server chỉ phản hồi khi @bot hoặc reply tin bot.

### Allowlists

| Field | Rỗng | Có giá trị |
| --- | --- | --- |
| `allowed_guilds` | Mọi guild | Chỉ guild IDs listed |
| `allowed_channels` | Mọi channel | Chỉ channel IDs listed |

### Test

```sh
curl -X POST http://127.0.0.1:18636/api/channels/discord/test
```

---

## Telegram

### Setup

1. [@BotFather](https://t.me/Botfather) → `/newbot` → lấy token
2. Set env: `VIETCLAW_TELEGRAM_TOKEN=...`
3. Enable config:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token_env": "VIETCLAW_TELEGRAM_TOKEN",
      "respond_in_groups": "mention_or_reply",
      "respond_in_private": true,
      "poll_timeout_seconds": 30
    }
  }
}
```

### Policy

| Setting | Mô tả |
| --- | --- |
| `respond_in_groups` | `mention_or_reply` / `always` / `never` |
| `respond_in_private` | Chat riêng với bot |
| `allowed_chats` | Allowlist chat IDs (rỗng = all) |

Trong group: bot phản hồi khi được mention hoặc reply.

### Test

```sh
curl -X POST http://127.0.0.1:18636/api/channels/telegram/test
```

---

## Sessions & idempotency

- Mỗi channel conversation map tới `sessions` table
- `channel_messages` lưu platform message ID — tránh xử lý duplicate
- `user_id` thường = platform user ID
- `channel` field = `discord` / `telegram`

Chat request nội bộ:

```json
{
  "message": "user text (mentions stripped)",
  "channel": "discord",
  "user_id": "123456789",
  "session_id": "..."
}
```

---

## Attachments

`channels.attachments` kiểm soát file đính kèm:

| Field | Default |
| --- | --- |
| `enabled` | `true` |
| `max_files` | `5` |
| `max_bytes` | `512 KB` |
| `allowed_extensions` | text-based extensions |

Attachments có thể được inject vào message context cho agent (xem `internal/channels/attachments.go`).

---

## Bảo mật channels

1. **Không add bot vào group công cộng** nếu `shell_exec` hoặc MCP nguy hiểm đang bật
2. Dùng `allowed_guilds` / `allowed_chats` whitelist
3. Giữ `mention_or_reply` trong group — tránh bot trả lời mọi tin
4. Token qua env — không commit vào `config.json`
5. Discord: tắt bot nếu không cần Message Content Intent scope rộng

---

## Multi-channel cùng user

Cùng một người trên Discord và web có `user_id` khác nhau → memory scope khác. Muốn shared memory, map `user_id` consistent hoặc dùng `memory_scope` chung trên agent profile.
