# Harness — coding task runner

Harness là subsystem chạy **coding tasks có cấu trúc**: lập plan, thực thi trong worktree, verify, trả diff. API mount tại `/api/harness/*`.

Phù hợp khi muốn agent sửa code có budget giới hạn, tool allowlist, và success checks rõ ràng — tách biệt với chat tự do.

---

## Lifecycle

```
POST /api/harness/runs     → status: planned
POST .../start             → running → localizing → patching → verifying
                              ↓
                    passed | failed | blocked | needs_approval | cancelled
POST .../cleanup           → dọn worktree
GET  .../diff              → git diff text
```

### Status values

| Status | Mô tả |
| --- | --- |
| `planned` | Đã tạo, chưa chạy |
| `running` | Đang thực thi |
| `localizing` | Setup workspace/worktree |
| `patching` | Agent đang sửa file |
| `verifying` | Chạy success checks |
| `passed` | Hoàn thành + checks OK |
| `failed` | Lỗi hoặc checks fail |
| `blocked` | Policy block |
| `needs_approval` | Cần human approval (budget/risk) |
| `cancelled` | User cancel |

---

## Tạo run

```sh
curl -s -X POST http://127.0.0.1:18636/api/harness/runs \
  -H 'Content-Type: application/json' \
  -d '{
    "goal": "Add health check endpoint to main.go",
    "workspace_root": "/home/user/myproject",
    "mode": "eco",
    "risk": "low",
    "auto_run": true,
    "max_usd": 0.50,
    "max_minutes": 20,
    "allowed_tools": ["file_read", "file_write", "file_grep"],
    "forbidden_tools": ["shell_exec"],
    "success_checks": ["go test ./...", "go vet ./..."]
  }'
```

### CreateRequest fields

| Field | Mô tả |
| --- | --- |
| `goal` | **Required** — mô tả task |
| `session_id` | Link tới chat session (optional) |
| `mode` | Runtime mode (`eco`, …) |
| `risk` | Risk level — ảnh hưởng approval |
| `workspace_root` | Repo root (default: agent workspace) |
| `auto_run` | `true` → start ngay sau create |
| `max_tokens` | Token budget |
| `max_usd` | USD budget |
| `max_minutes` | Time budget |
| `allowed_tools` | Tool allowlist cho run này |
| `forbidden_tools` | Tool denylist |
| `success_checks` | Shell commands verify kết quả |

---

## Plan structure

Mỗi run có `plan_json` chứa:

```json
{
  "goal": "...",
  "mode": "eco",
  "risk": "low",
  "summary": "...",
  "assumptions": ["..."],
  "steps": [
    {
      "id": "step-1",
      "title": "Read existing code",
      "description": "...",
      "tools": ["file_read"],
      "checks": []
    }
  ],
  "stop_rules": ["Do not modify vendor/", "..."]
}
```

---

## Worktree isolation

Harness có thể tạo **git worktree** riêng:

| Field | Mô tả |
| --- | --- |
| `worktree_path` | Path worktree tạm |
| `branch_name` | Branch làm việc |
| `base_ref` | Ref gốc (main, HEAD, …) |
| `changed_files` | Files đã sửa |
| `final_diff` | Unified diff cuối |

`POST .../cleanup` xóa worktree/branch sau khi review.

---

## Events

`harness_events` log từng bước — `GET /api/harness/runs/{id}` trả `run` + `events[]`.

Dùng để debug plan execution, tool calls trong harness context.

---

## API summary

| Method | Path | Action |
| --- | --- | --- |
| GET | `/api/harness/runs` | List runs |
| POST | `/api/harness/runs` | Create |
| GET | `/api/harness/runs/{id}` | Detail + events |
| POST | `/api/harness/runs/{id}/start` | Start |
| POST | `/api/harness/runs/{id}/cancel` | Cancel |
| GET | `/api/harness/runs/{id}/diff` | Plain text diff |
| POST | `/api/harness/runs/{id}/cleanup` | Cleanup worktree |

---

## Khi nào dùng harness vs chat

| | Chat (`/api/chat`) | Harness |
| --- | --- | --- |
| Use case | Hỏi đáp, automation linh hoạt | Sửa code có verify |
| Budget | Global daily limit | Per-run budget |
| Tools | Profile/global policy | Per-run allow/deny |
| Output | Text reply | Git diff + status |
| Isolation | Workspace chung | Worktree riêng |

---

## Lưu ý

- `workspace_root` phải là git repo nếu dùng worktree features
- `success_checks` chạy shell — cân nhắc security tương tự `shell_exec`
- Review `final_diff` trước merge — harness không auto-push
- Web UI có thể expose harness qua advanced console (nếu enabled)
