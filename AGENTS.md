# AGENTS.md

## Cursor Cloud specific instructions

VietClaw is a single Go binary (`cmd/vietclaw`) that serves an HTTP API + an embedded Nuxt 4 SPA, backed by SQLite. The web UI source lives in `apps/web` and is built into `internal/web/dist` (embedded via `//go:embed`). Standard commands live in `README.md` (Development section) and `.github/workflows/ci.yml`.

### Services
- **Backend daemon** — `go run ./cmd/vietclaw daemon` (or build first, see below). Serves API + embedded UI at `http://127.0.0.1:18636`. Run `./vietclaw init` once to create the data dir/config/DB (`~/.vietclaw/`, override with `VIETCLAW_DATA_DIR`).
- **Web UI** — embedded into the binary at build time. For live iteration, run `pnpm dev` in `apps/web` (separate port) which proxies to the daemon.

### Non-obvious gotchas
- **Build the web UI before the Go binary serves a current UI.** `internal/web/dist/index.html` is committed but references hashed JS/CSS assets that are gitignored. The update script does NOT build the UI. To get a working/up-to-date embedded UI you MUST run `pnpm build` in `apps/web` (this generates `.output/public` and copies it to `internal/web/dist/`), THEN `go build ./cmd/vietclaw` (or `go run`). The Go binary embeds whatever is in `internal/web/dist` at compile time, so rebuild the Go binary after rebuilding the web UI.
- **No external services needed.** SQLite only; no Postgres/Redis/Docker. Backend runs standalone.
- **Mock provider works with zero config**, so chat is testable end-to-end without any API keys. Real providers need env vars (`OPENAI_API_KEY`, `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, `OPENCODE_ZEN_KEY`) — see README Configuration.
- **Quick chat smoke test:** `curl -s -X POST http://127.0.0.1:18636/api/chat -H 'Content-Type: application/json' -d '{"message":"hi","channel":"web","user_id":"test"}'`. Health: `GET /health`. Note the API mounts under `/api/*` and bare `/health`, `/status` (not `/api/health`).
- **`pnpm typecheck` currently fails** with pre-existing TypeScript errors in `app/components/ChatPanel.vue` / `app/composables/useChat.ts` (unrelated to environment setup). `pnpm build` succeeds regardless because Nuxt's static generate does not typecheck.
- **Go version:** `go.mod` requires Go 1.25; the toolchain auto-downloads it if needed.

### Lint / test / build commands
- Go tests: `go test ./...` (some `tests/channels` take ~10s).
- Go vet: `go vet ./...`.
- Web build (refresh embedded UI): `cd apps/web && pnpm build`.
- Full binary: `go build -o vietclaw ./cmd/vietclaw`.
