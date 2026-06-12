# Getting started

## Yêu cầu

| Thành phần | Phiên bản |
| --- | --- |
| Go | 1.25+ |
| Node.js | 24+ (chỉ khi build/dev web UI) |
| pnpm | 10.33+ |

Không cần Docker, Postgres, Redis. Mock provider chạy ngay không cần API key.

## Cài đặt nhanh

```sh
git clone https://github.com/vietclaw/vietclaw.git
cd vietclaw

go run ./cmd/vietclaw init
go run ./cmd/vietclaw daemon
```

Mở **http://127.0.0.1:18636** — chat ngay với mock provider.

## Thư mục dữ liệu

Mặc định: `~/.vietclaw/` (Windows: `%USERPROFILE%\.vietclaw\`).

Override bằng biến môi trường `VIETCLAW_DATA_DIR`.

| Path | Mô tả |
| --- | --- |
| `config.json` | Cấu hình chính |
| `vietclaw.db` | SQLite (sessions, memory, runs, costs…) |
| `workspace/` | Workspace cho file tools |
| `logs/vietclaw.log` | Log file daemon |

## CLI commands

| Lệnh | Mô tả |
| --- | --- |
| `vietclaw init` | Tạo data dir, config mặc định, database |
| `vietclaw setup` | Wizard cấu hình tương tác (provider, channels) |
| `vietclaw daemon` | Chạy HTTP server + channels |
| `vietclaw status` | Truy vấn daemon đang chạy |
| `vietclaw doctor` | Health checks (config, DB, migration) |
| `vietclaw websearch` | Quản lý tích hợp open-websearch |

## Cấu hình provider thật

Đặt API key qua env (hoặc `.env` trong cwd / data dir):

```sh
export OPENAI_API_KEY=sk-...
export GEMINI_API_KEY=...
export ANTHROPIC_API_KEY=...
export OPENCODE_ZEN_KEY=...
```

Sau đó chỉnh `config.json` hoặc dùng `vietclaw setup` / web UI **Công cụ nâng cao**.

## Smoke test (API)

```sh
# Health
curl -s http://127.0.0.1:18636/health

# Chat đồng bộ
curl -s -X POST http://127.0.0.1:18636/api/chat \
  -H 'Content-Type: application/json' \
  -d '{"message":"xin chào","channel":"web","user_id":"test"}'

# Status daemon
curl -s http://127.0.0.1:18636/status
```

## Build production binary

Web UI được embed vào binary Go lúc compile — **phải build web trước**:

```sh
cd apps/web && pnpm install && pnpm build && cd ../..
go build -o vietclaw ./cmd/vietclaw
./vietclaw init
./vietclaw daemon
```

`internal/web/dist` chỉ commit `index.html`; JS/CSS hashed bị gitignore. Binary embed đúng nội dung trong `dist` tại thời điểm `go build`.

## Dev với hot reload UI

```sh
go run ./cmd/vietclaw daemon          # backend :18636
cd apps/web && pnpm dev               # Nuxt dev server, proxy API
```

## Bước tiếp theo

- [Configuration](configuration.md) — toàn bộ `config.json`
- [HTTP API](api.md) — tích hợp programmatic
- [Agents & framework](agents.md) — multi-agent profiles
