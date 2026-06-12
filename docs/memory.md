# Memory system

Long-term memory lưu trong SQLite (`memories` table), recall hybrid FTS + vector embedding.

## Data model

```sql
memories (
  id, scope, kind, content, confidence,
  created_at, updated_at, embedding BLOB
)
```

### Kinds

| Kind | Dùng cho |
| --- | --- |
| `profile` | Thông tin user (tên, role, …) |
| `preference` | Sở thích, style |
| `project` | Context dự án |
| `workflow` | Quy trình làm việc |
| `decision` | Quyết định đã chốt |
| `connection` | Server, API, credentials ref |
| `experience` | Lessons (reflexion, learned errors) |
| `note` | Ghi chú chung |

### Confidence

| Label | Score | Ý nghĩa |
| --- | --- | --- |
| `confirmed` | 1.0 | User xác nhận / explicit store |
| `inferred` | 0.7 | Agent suy luận |
| `temporary` | 0.35 | Có thể hết hạn / thử nghiệm |

---

## Scopes

Memory isolated theo `scope` string — thường = `user_id` từ chat request.

Agent profile có thể override qua `memory_scope` — ví dụ team shared scope `"team-research"`.

```
user "alice"  → scope "alice"
agent profile memory_scope "shared" → scope "shared"
```

---

## Recall — hybrid search

`SearchHybrid(scope, query, limit, embedder)`:

1. **Keyword (FTS)** — SQLite full-text search, lấy `limit * 3` candidates
2. **Vector** — nếu có embedder + query không rỗng: cosine similarity trên `embedding` BLOB
3. **Merge & rank** — kết hợp scores, trả top `limit`

Không có embedder → chỉ FTS.

### Context injection

Mỗi chat, builder lấy top 9 memories cho user message:

- 6 core memories (profile, preference, project, …)
- 3 experience/lesson memories

Format trong system prompt:

```
[Memory header]
- {content}
- [lesson] {experience content}
```

---

## Cách memory được tạo

| Nguồn | Cơ chế |
| --- | --- |
| User nói "nhớ là …" | Intent `memory_add` → rule handler |
| Agent tool `memory_store` | Trong agentic loop |
| Reflexion | Tool fail → auto `experience` |
| API `POST /api/memory` | Manual / integration |
| Web UI Memory view | CRUD qua API |

---

## API operations

Xem [HTTP API — Memory](api.md#memory).

| Operation | Endpoint |
| --- | --- |
| List | `GET /api/memory?scope=` |
| Add | `POST /api/memory` |
| Search | `GET /api/memory/search?scope=&q=` |
| Delete | `DELETE /api/memory/{id}` |
| Curate duplicates | `POST /api/memory/curate?scope=` |

### Curate

`CurateDuplicates(scope)` — gộp/xóa memories trùng lặp trong scope. Hữu ích sau nhiều session import facts giống nhau.

---

## Memory tools (agent-facing)

Khi `agent.memory_tools.enabled`:

**`memory_recall`** — agent chủ động search khi cần context không có trong auto-inject.

**`memory_store`** — agent lưu fact với kind + confidence tùy chọn.

---

## Embedding provider

Embedding tạo lúc:

- `POST /api/memory` (add manual)
- `memory_store` tool
- reflexion lesson

Cần provider có `embed_model` configured. Không có embedder → memory vẫn hoạt động FTS-only.

---

## Best practices

1. **Scope rõ ràng** — dùng `user_id` consistent across channels
2. **Kind đúng** — giúp ranking và UI filter
3. **Confirmed vs inferred** — đừng promote inferred lên confirmed tự động
4. **Curate định kỳ** — tránh duplicate noise trong context
5. **Experience lessons** — để reflexion chạy; review và xóa lesson sai qua UI
