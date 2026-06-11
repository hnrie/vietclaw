# Contributing to VietClaw

Thank you for your interest in contributing. This document explains how to get started, what we expect in pull requests, and how issues are handled.

## Table of contents

- [Before you start](#before-you-start)
- [Development setup](#development-setup)
- [Making changes](#making-changes)
- [Testing](#testing)
- [Pull requests](#pull-requests)
- [Issue guidelines](#issue-guidelines)
- [Code style](#code-style)
- [Community](#community)

## Before you start

- Read the [README](README.md) for project scope and architecture.
- Check [open issues](https://github.com/vietclaw/vietclaw/issues) and [pull requests](https://github.com/vietclaw/vietclaw/pulls) to avoid duplicate work.
- For large features, open an issue or discussion first so we can align on design.
- Security issues: see [SECURITY.md](SECURITY.md) — do not file public issues.

## Development setup

**Requirements**

| Component | Version |
| --- | --- |
| Go | 1.25+ |
| Node.js | 24+ (web UI only) |
| pnpm | 10.33+ (see `apps/web/package.json`) |

**Clone and initialize**

```sh
git clone https://github.com/vietclaw/vietclaw.git
cd vietclaw

go run ./cmd/vietclaw init

# Build embedded web UI (required for go build / full UI)
cd apps/web
pnpm install
pnpm build
cd ../..
```

**Run the daemon**

```sh
go run ./cmd/vietclaw daemon
```

**Web UI dev server** (hot reload, proxies API to daemon):

```sh
cd apps/web && pnpm dev
```

**Environment**

- API keys via `.env` in project root or `%APPDATA%\VietClaw\.env` (Windows) / `~/.vietclaw/.env` (Unix).
- Mock provider works without keys for basic agent loop testing.

## Making changes

1. Fork the repository and create a branch from `master`:

   ```sh
   git checkout -b feat/my-feature
   ```

2. Keep changes focused — one logical change per PR when possible.

3. Update documentation if you change user-facing behavior, CLI, config schema, or HTTP APIs.

4. Do **not** commit `internal/web/dist` build output (except the stub `index.html`). CI builds the UI before tests.

5. Do not commit secrets, `.env` files with real keys, or local databases.

## Testing

```sh
# Full suite (mirror CI: build web first)
cd apps/web && pnpm install --frozen-lockfile && pnpm build && cd ../..
go test ./...

# Web typecheck
cd apps/web && pnpm typecheck
```

CI runs on every push to `master`, `main`, `dev`, and `unstable`, and on all pull requests ([`.github/workflows/ci.yml`](workflows/ci.yml)).

## Pull requests

1. Ensure tests pass locally.
2. Write a clear PR title and description:
   - **What** changed and **why**
   - Link related issues (`Fixes #123`)
   - Note breaking changes or config migrations
3. Keep PRs reasonably sized; split large work when possible.
4. Maintainers may request changes or squash commits on merge.

We do not require a CLA at this time. By contributing, you agree that your contributions are licensed under the same [MIT License](LICENSE) as the project.

## Issue guidelines

Use the appropriate [issue template](ISSUE_TEMPLATE/):

| Template | Use when |
| --- | --- |
| Bug report | Something broke or regressed |
| Feature request | New capability or enhancement |

Include reproduction steps for bugs, and motivation + alternatives for features. Incomplete reports may be closed if we cannot reproduce or scope the work.

## Code style

**Go**

- Match existing patterns in `internal/` — minimal scope, clear names, no unnecessary abstraction.
- Run `go fmt` / `go vet` on changed packages.
- Prefer table-driven tests in `tests/`.

**Web (`apps/web`)**

- Vue 3 Composition API + TypeScript.
- Tailwind utility classes; follow existing component structure.
- User-facing strings: add keys to `apps/web/locales/vi.json` and `en.json`.

**i18n (backend)**

- Strings live in `internal/i18n/locales/*.json`, not inline in Go source.

## Community

- [Code of Conduct](CODE_OF_CONDUCT.md)
- Be respectful and constructive in issues, reviews, and discussions.

Questions welcome in issues when they are specific and actionable. For broad brainstorming, label your issue clearly so maintainers can triage.
